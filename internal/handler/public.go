package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

type PublicHandler struct {
	queries *repository.Queries
	logger  *slog.Logger
}

func NewPublicHandler(queries *repository.Queries, logger *slog.Logger) *PublicHandler {
	return &PublicHandler{
		queries: queries,
		logger:  logger,
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

	// Save lead to database (best-effort, don't fail the request)
	ip := r.RemoteAddr
	ua := r.UserAgent()
	score := int32(req.MockScore)
	_, err := h.queries.InsertAuditLead(r.Context(), repository.InsertAuditLeadParams{
		IgUsername: username,
		IpAddress:  &ip,
		UserAgent:  &ua,
		Locale:     strPtr(req.Locale),
		MockScore:  &score,
	})
	if err != nil {
		h.logger.Warn("audit: failed to save lead",
			slog.String("ig_username", username),
			slog.String("error", err.Error()),
		)
	} else {
		h.logger.Info("audit: lead saved",
			slog.String("ig_username", username),
			slog.String("remote_addr", r.RemoteAddr),
		)
	}

	// Return total count for social proof
	total, _ := h.queries.CountAuditLeads(r.Context())

	respondJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"total":  total,
	})
}

func (h *PublicHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	total, _ := h.queries.CountAuditLeads(r.Context())
	respondJSON(w, http.StatusOK, map[string]any{
		"status": "demo",
		"total":  total,
	})
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
