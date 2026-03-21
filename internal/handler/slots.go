package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type SlotHandler struct {
	slotSvc *service.SlotService
	logger  *slog.Logger
}

func NewSlotHandler(slotSvc *service.SlotService, logger *slog.Logger) *SlotHandler {
	return &SlotHandler{slotSvc: slotSvc, logger: logger}
}

func (h *SlotHandler) List(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	slots, err := h.slotSvc.ListSlots(r.Context(), planID)
	if err != nil {
		h.logger.Error("list slots failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list slots")
		return
	}

	respondJSON(w, http.StatusOK, slots)
}

func (h *SlotHandler) Get(w http.ResponseWriter, r *http.Request) {
	slotID, ok := parseUUID(chi.URLParam(r, "slot_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid slot ID")
		return
	}

	slot, err := h.slotSvc.GetSlot(r.Context(), slotID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "slot not found")
		return
	}

	respondJSON(w, http.StatusOK, slot)
}

func (h *SlotHandler) Update(w http.ResponseWriter, r *http.Request) {
	slotID, ok := parseUUID(chi.URLParam(r, "slot_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid slot ID")
		return
	}

	var req dto.UpdateSlotRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	slot, err := h.slotSvc.UpdateSlot(r.Context(), slotID, req)
	if err != nil {
		h.logger.Error("update slot failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update slot")
		return
	}

	respondJSON(w, http.StatusOK, slot)
}

func (h *SlotHandler) ApproveAll(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	result, err := h.slotSvc.ApproveAll(r.Context(), planID)
	if err != nil {
		h.logger.Error("approve all failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to approve slots")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *SlotHandler) StartPosting(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	result, err := h.slotSvc.StartPosting(r.Context(), planID)
	if err != nil {
		h.logger.Error("start posting failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to start posting")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
