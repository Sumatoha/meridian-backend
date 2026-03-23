package handler

import (
	"log/slog"
	"net/http"

	"github.com/meridian/api/internal/dto"
)

type PublicHandler struct {
	logger *slog.Logger
}

func NewPublicHandler(logger *slog.Logger) *PublicHandler {
	return &PublicHandler{logger: logger}
}

func (h *PublicHandler) StartAudit(w http.ResponseWriter, r *http.Request) {
	var req dto.PublicAuditRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.IGUsername == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "ig_username is required")
		return
	}

	h.logger.Info("audit: demo requested",
		slog.String("ig_username", req.IGUsername),
		slog.String("remote_addr", r.RemoteAddr),
	)

	// Public audit uses client-side mock data — no server processing needed.
	// This endpoint exists only for analytics/logging purposes.
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "demo",
	})
}

func (h *PublicHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, dto.PublicAuditResponse{
		Status: "demo",
	})
}
