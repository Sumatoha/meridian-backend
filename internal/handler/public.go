package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

// PublicAnalyzer is the interface for running public (unauthenticated) profile analysis.
type PublicAnalyzer interface {
	AnalyzePublic(ctx context.Context, username string) (dto.PublicAuditResult, error)
}

type PublicHandler struct {
	analyzer PublicAnalyzer
	queries  *repository.Queries
	logger   *slog.Logger
}

func NewPublicHandler(analyzer PublicAnalyzer, queries *repository.Queries, logger *slog.Logger) *PublicHandler {
	return &PublicHandler{
		analyzer: analyzer,
		queries:  queries,
		logger:   logger,
	}
}

func (h *PublicHandler) StartAudit(w http.ResponseWriter, r *http.Request) {
	var req dto.PublicAuditRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	username := strings.TrimSpace(req.IGUsername)
	username = strings.TrimPrefix(username, "@")

	if username == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "ig_username is required")
		return
	}

	h.logger.Info("audit: started",
		slog.String("ig_username", username),
		slog.String("remote_addr", r.RemoteAddr),
	)

	// Run real analysis
	result, err := h.analyzer.AnalyzePublic(r.Context(), username)
	if err != nil {
		h.logger.Error("audit: analysis failed",
			slog.String("ig_username", username),
			slog.String("error", err.Error()),
		)

		// Save lead even on failure (best-effort)
		h.saveLead(r, username, req.Locale, 0)

		respondError(w, http.StatusUnprocessableEntity, "analysis_failed",
			"Could not analyze this profile. Please make sure it's public.")
		return
	}

	result.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save lead with real score
	h.saveLead(r, username, req.Locale, result.Score)

	respondJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *PublicHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	total, _ := h.queries.CountAuditLeads(r.Context())
	respondJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"total":  total,
	})
}

func (h *PublicHandler) saveLead(r *http.Request, username, locale string, score int) {
	ip := r.RemoteAddr
	ua := r.UserAgent()
	s := int32(score)
	_, err := h.queries.InsertAuditLead(r.Context(), repository.InsertAuditLeadParams{
		IgUsername: username,
		IpAddress:  &ip,
		UserAgent:  &ua,
		Locale:     strPtr(locale),
		MockScore:  &s,
	})
	if err != nil {
		h.logger.Warn("audit: failed to save lead",
			slog.String("ig_username", username),
			slog.String("error", err.Error()),
		)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
