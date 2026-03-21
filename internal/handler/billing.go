package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type BillingHandler struct {
	svc    *service.BillingService
	logger *slog.Logger
}

func NewBillingHandler(svc *service.BillingService, logger *slog.Logger) *BillingHandler {
	return &BillingHandler{svc: svc, logger: logger}
}

func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req dto.CheckoutRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.Plan == "" || req.Provider == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "plan and provider are required")
		return
	}

	userID := auth.UserID(r.Context())
	url, err := h.svc.CreateCheckout(r.Context(), userID, req)
	if err != nil {
		h.logger.Error("checkout failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create checkout")
		return
	}

	respondJSON(w, http.StatusOK, dto.CheckoutResponse{CheckoutURL: url})
}

func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	sub, err := h.svc.GetSubscription(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get subscription")
		return
	}

	respondJSON(w, http.StatusOK, sub)
}

func (h *BillingHandler) DodoWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "failed to read body")
		return
	}
	defer r.Body.Close()

	if err := h.svc.HandleDodoWebhook(r.Context(), body); err != nil {
		h.logger.Error("dodo webhook failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "webhook processing failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BillingHandler) KaspiWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "failed to read body")
		return
	}
	defer r.Body.Close()

	if err := h.svc.HandleKaspiWebhook(r.Context(), body); err != nil {
		h.logger.Error("kaspi webhook failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "webhook processing failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}
