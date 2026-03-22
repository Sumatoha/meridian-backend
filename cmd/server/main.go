package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"encoding/json"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meridian/api/internal/ai"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/config"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/handler"
	"github.com/meridian/api/internal/instagram"
	"github.com/meridian/api/internal/middleware"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/scraper"
	"github.com/meridian/api/internal/service"
	"github.com/meridian/api/internal/storage"
)

func maskDSN(dsn string) string {
	// Show host only, hide credentials
	if idx := strings.Index(dsn, "@"); idx != -1 {
		rest := dsn[idx:]
		if end := strings.Index(rest, "/"); end != -1 {
			return "***" + rest[:end]
		}
		return "***" + rest
	}
	return "***"
}

func main() {
	// Structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("server version", slog.String("build", "2026-03-23-v3-batch-gen"))

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("database connected", slog.String("host", maskDSN(cfg.DatabaseURL)))

	// Initialize repository
	queries := repository.New(pool)

	// Initialize external clients
	aiClient := ai.NewClient(cfg.AnthropicAPIKey)
	igScraper := scraper.NewScraper(
		os.Getenv("RAPIDAPI_KEY"),
		os.Getenv("RAPIDAPI_BASE_URL"),
	)
	igPublisher := instagram.NewPublisher(cfg.MetaAppID, cfg.MetaAppSecret)
	igOAuthClient := instagram.NewOAuthClient(cfg.MetaAppID, cfg.MetaAppSecret, cfg.MetaOAuthRedirectURI)
	igReader := instagram.NewReader()
	storageClient := storage.NewClient(cfg.SupabaseStorageURL, cfg.SupabaseServiceKey)

	// Initialize services
	accountSvc := service.NewAccountService(pool, queries, igOAuthClient, cfg.MetaAppSecret)
	analysisSvc := service.NewAnalysisService(queries, aiClient, igScraper, igReader, logger)
	planSvc := service.NewPlanService(queries, aiClient, logger)
	slotSvc := service.NewSlotService(queries, storageClient)
	publisherSvc := service.NewPublisherService(queries, igPublisher, storageClient, logger)
	billingSvc := service.NewBillingService(queries, cfg.DodoAPIKey, cfg.KaspiMerchantID, cfg.KaspiSecret)

	_ = publisherSvc // Used by River jobs

	// Initialize handlers
	accountH := handler.NewAccountHandler(accountSvc, logger)
	settingsH := handler.NewSettingsHandler(queries, logger)
	analysisH := handler.NewAnalysisHandler(analysisSvc, accountSvc, logger)
	planH := handler.NewPlanHandler(planSvc, accountSvc, logger)
	slotH := handler.NewSlotHandler(slotSvc, logger)
	mediaH := handler.NewMediaHandler(slotSvc, storageClient, queries, logger)
	billingH := handler.NewBillingHandler(billingSvc, logger)
	publicH := handler.NewPublicHandler(logger)

	// Auth middleware — JWKS (RS256) with HMAC fallback
	jwksURL := ""
	if cfg.SupabaseURL != "" {
		jwksURL = strings.TrimRight(cfg.SupabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
	}
	authMW, err := auth.NewMiddleware(jwksURL, cfg.SupabaseJWTSecret)
	if err != nil {
		logger.Error("failed to initialize auth middleware", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Rate limiters
	aiRateLimiter := middleware.NewRateLimiter(5, time.Minute)
	publicRateLimiter := middleware.NewRateLimiter(3, time.Hour)

	// User resolution middleware: upserts user on every authenticated request
	userResolver := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			supabaseUID := auth.SupabaseUserID(r.Context())
			if supabaseUID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			userID, err := accountSvc.EnsureUser(r.Context(), supabaseUID, "")
			if err != nil {
				logger.Error("user resolution failed",
					slog.String("error", err.Error()),
					slog.String("supabase_uid", supabaseUID.String()),
					slog.String("path", r.URL.Path),
				)
				http.Error(w, `{"error":{"code":"internal_error","message":"user resolution failed"}}`, http.StatusInternalServerError)
				return
			}

			ctx := auth.SetUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Router
	r := chi.NewRouter()

	// Global middleware
	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(allowedOrigin))
	r.Use(middleware.Logging(logger))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := pool.Ping(r.Context()); err != nil {
			dbStatus = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.HealthResponse{
			Status: "ok",
			DB:     dbStatus,
		})
	})

	// Public routes (no auth)
	r.Route("/api/v1/public", func(r chi.Router) {
		r.Use(publicRateLimiter.Middleware(func(r *http.Request) string {
			return r.RemoteAddr
		}))
		r.Post("/audit", publicH.StartAudit)
		r.Get("/audit/{job_id}", publicH.GetAudit)
	})

	// Billing webhooks (no auth, verified by signature)
	r.Post("/api/v1/billing/webhook/dodo", billingH.DodoWebhook)
	r.Post("/api/v1/billing/webhook/kaspi", billingH.KaspiWebhook)

	// Authenticated routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMW.Authenticate)
		r.Use(userResolver)

		// Accounts — OAuth routes first (before {id} param)
		r.Get("/accounts/oauth/url", accountH.GetOAuthURL)
		r.Post("/accounts/oauth/callback", accountH.OAuthCallback)
		r.Post("/accounts", accountH.Create)
		r.Get("/accounts", accountH.List)
		r.Get("/accounts/{id}", accountH.Get)
		r.Delete("/accounts/{id}", accountH.Delete)

		// Brand settings
		r.Get("/accounts/{account_id}/settings", settingsH.Get)
		r.Put("/accounts/{account_id}/settings", settingsH.Update)

		// Analysis (rate limited)
		r.Group(func(r chi.Router) {
			r.Use(aiRateLimiter.Middleware(func(r *http.Request) string {
				return auth.UserID(r.Context()).String()
			}))
			r.Post("/accounts/{account_id}/analyze", analysisH.Analyze)
			r.Post("/accounts/{account_id}/plans/generate", planH.Generate)
		})

		r.Get("/accounts/{account_id}/analysis", analysisH.GetAnalysis)

		// Plans
		r.Get("/accounts/{account_id}/plans", planH.List)
		r.Get("/plans/{plan_id}", planH.Get)
		r.Patch("/plans/{plan_id}", planH.Update)
		r.Delete("/plans/{plan_id}", planH.Delete)

		// Slots
		r.Get("/plans/{plan_id}/slots", slotH.List)
		r.Get("/slots/{slot_id}", slotH.Get)
		r.Patch("/slots/{slot_id}", slotH.Update)

		// Media
		r.Post("/slots/{slot_id}/media", mediaH.Upload)
		r.Delete("/slots/{slot_id}/media/{index}", mediaH.Delete)

		// Plan actions
		r.Post("/plans/{plan_id}/approve-all", slotH.ApproveAll)
		r.Post("/plans/{plan_id}/start-posting", slotH.StartPosting)

		// Billing
		r.Post("/billing/checkout", billingH.Checkout)
		r.Get("/billing/subscription", billingH.GetSubscription)
	})

	logger.Info("services initialized",
		slog.Bool("has_anthropic_key", cfg.AnthropicAPIKey != ""),
		slog.Bool("has_meta_app", cfg.MetaAppID != ""),
		slog.String("meta_redirect_uri", cfg.MetaOAuthRedirectURI),
		slog.Bool("has_dodo", cfg.DodoAPIKey != ""),
		slog.Bool("has_kaspi", cfg.KaspiMerchantID != ""),
		slog.String("cors_origin", allowedOrigin),
	)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("server starting", slog.String("addr", addr), slog.String("env", cfg.Environment))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced shutdown", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
