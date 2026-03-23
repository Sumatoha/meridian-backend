package handler

import (
	"log/slog"
	"net/http"

	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/service"
)

type TierHandler struct {
	tierSvc *service.TierService
	logger  *slog.Logger
}

func NewTierHandler(tierSvc *service.TierService, logger *slog.Logger) *TierHandler {
	return &TierHandler{tierSvc: tierSvc, logger: logger}
}

func (h *TierHandler) GetTierInfo(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())

	info, err := h.tierSvc.GetTierInfo(r.Context(), userID)
	if err != nil {
		h.logger.Error("get tier info failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get tier info")
		return
	}

	respondJSON(w, http.StatusOK, info)
}
