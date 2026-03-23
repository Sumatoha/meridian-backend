package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type SharingHandler struct {
	sharingSvc *service.SharingService
	logger     *slog.Logger
}

func NewSharingHandler(sharingSvc *service.SharingService, logger *slog.Logger) *SharingHandler {
	return &SharingHandler{sharingSvc: sharingSvc, logger: logger}
}

// CreateShareLink generates a share token for the plan.
func (h *SharingHandler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	userID := auth.UserID(r.Context())
	token, err := h.sharingSvc.CreateShareLink(r.Context(), userID, planID)
	if err != nil {
		var tierErr *service.TierError
		if errors.As(err, &tierErr) {
			respondJSON(w, http.StatusForbidden, dto.TierLimitError{
				Code:      "tier_limit",
				Message:   tierErr.Message,
				Feature:   tierErr.Feature,
				UpgradeTo: tierErr.UpgradeTo,
			})
			return
		}
		h.logger.Error("create share link failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create share link")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}

// RevokeShareLink removes the share token.
func (h *SharingHandler) RevokeShareLink(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	if err := h.sharingSvc.RevokeShareLink(r.Context(), planID); err != nil {
		h.logger.Error("revoke share link failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to revoke share link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSharedPlan returns a plan by its public share token (no auth required).
func (h *SharingHandler) GetSharedPlan(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "missing_token", "share token is required")
		return
	}

	plan, err := h.sharingSvc.GetSharedPlan(r.Context(), token)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "shared plan not found")
		return
	}

	respondJSON(w, http.StatusOK, plan)
}
