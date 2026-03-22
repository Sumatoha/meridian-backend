package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	supabaseKey contextKey = "supabase_user_id"
)

type Middleware struct {
	jwks    keyfunc.Keyfunc
	hmacKey []byte // fallback for legacy HS256 tokens
}

// NewMiddleware creates an auth middleware that validates JWTs using JWKS (RS256)
// with a fallback to HMAC (HS256) for legacy Supabase tokens.
//
// jwksURL: e.g. "https://<ref>.supabase.co/.well-known/jwks.json"
// legacySecret: the old SUPABASE_JWT_SECRET (can be empty to disable HMAC fallback)
func NewMiddleware(jwksURL string, legacySecret string) (*Middleware, error) {
	m := &Middleware{}

	// Set up JWKS for RS256 validation
	if jwksURL != "" {
		jwks, err := keyfunc.NewDefault([]string{jwksURL})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JWKS from %s: %w", jwksURL, err)
		}
		m.jwks = jwks
		slog.Info("auth: JWKS initialized", slog.String("jwks_url", jwksURL))
	}

	// Set up HMAC fallback for legacy tokens
	if legacySecret != "" {
		cleaned := strings.TrimSpace(legacySecret)
		if len(cleaned) != len(legacySecret) {
			slog.Warn("jwt secret had leading/trailing whitespace — trimmed",
				slog.Int("original_len", len(legacySecret)),
				slog.Int("cleaned_len", len(cleaned)),
			)
		}
		m.hmacKey = []byte(cleaned)
		slog.Info("auth: HMAC fallback enabled", slog.Int("secret_len", len(cleaned)))
	}

	if m.jwks == nil && len(m.hmacKey) == 0 {
		return nil, fmt.Errorf("at least one of JWKS URL or legacy JWT secret must be provided")
	}

	return m, nil
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			slog.Warn("auth: missing authorization header",
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"missing authorization header"}}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			slog.Warn("auth: invalid authorization format (missing Bearer prefix)",
				slog.String("path", r.URL.Path),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid authorization format"}}`, http.StatusUnauthorized)
			return
		}

		token, err := m.parseToken(tokenStr)
		if err != nil || !token.Valid {
			errMsg := "unknown"
			if err != nil {
				errMsg = err.Error()
			}
			slog.Warn("auth: jwt verification failed",
				slog.String("error", errMsg),
				slog.String("path", r.URL.Path),
				slog.Int("token_len", len(tokenStr)),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid or expired token"}}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.Warn("auth: invalid token claims type", slog.String("path", r.URL.Path))
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid token claims"}}`, http.StatusUnauthorized)
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			slog.Warn("auth: missing subject in token", slog.String("path", r.URL.Path))
			http.Error(w, `{"error":{"code":"unauthorized","message":"missing subject in token"}}`, http.StatusUnauthorized)
			return
		}

		supabaseUID, err := uuid.Parse(sub)
		if err != nil {
			slog.Warn("auth: invalid subject format",
				slog.String("sub", sub),
				slog.String("path", r.URL.Path),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid subject format"}}`, http.StatusUnauthorized)
			return
		}

		slog.Debug("auth: authenticated",
			slog.String("uid", supabaseUID.String()),
			slog.String("path", r.URL.Path),
			slog.String("alg", token.Method.Alg()),
		)

		ctx := context.WithValue(r.Context(), supabaseKey, supabaseUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// parseToken tries JWKS first (RS256/ES256), then falls back to HMAC (HS256).
func (m *Middleware) parseToken(tokenStr string) (*jwt.Token, error) {
	// Try JWKS first (asymmetric keys — RS256, ES256, EdDSA)
	if m.jwks != nil {
		token, err := jwt.Parse(tokenStr, m.jwks.KeyfuncCtx(context.Background()),
			jwt.WithValidMethods([]string{"RS256", "ES256", "EdDSA"}),
		)
		if err == nil && token.Valid {
			return token, nil
		}
		slog.Debug("auth: JWKS validation failed, trying HMAC fallback",
			slog.String("error", err.Error()),
		)
	}

	// Fallback to HMAC (legacy HS256 tokens)
	if len(m.hmacKey) > 0 {
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
			}
			return m.hmacKey, nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
		if err == nil && token.Valid {
			return token, nil
		}
		return token, err
	}

	return nil, fmt.Errorf("no valid signing method found for token")
}

func SupabaseUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(supabaseKey).(uuid.UUID)
	return id
}

func SetUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey).(uuid.UUID)
	return id
}
