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

	// TODO: Enqueue limited audit job
	// For now, return a placeholder job ID
	respondJSON(w, http.StatusAccepted, map[string]string{
		"job_id": "placeholder",
		"status": "pending",
	})
}

func (h *PublicHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement audit result retrieval
	respondJSON(w, http.StatusOK, dto.PublicAuditResponse{
		Status: "pending",
	})
}
