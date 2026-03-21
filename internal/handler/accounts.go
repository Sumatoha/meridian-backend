package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type AccountHandler struct {
	svc    *service.AccountService
	logger *slog.Logger
}

func NewAccountHandler(svc *service.AccountService, logger *slog.Logger) *AccountHandler {
	return &AccountHandler{svc: svc, logger: logger}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.IGUsername == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "ig_username is required")
		return
	}

	account, err := h.svc.CreateAccount(r.Context(), req)
	if err != nil {
		h.logger.Error("create account failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create account")
		return
	}

	respondJSON(w, http.StatusCreated, account)
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.svc.ListAccounts(r.Context())
	if err != nil {
		h.logger.Error("list accounts failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list accounts")
		return
	}

	respondJSON(w, http.StatusOK, accounts)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(chi.URLParam(r, "id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	if err := h.svc.DeleteAccount(r.Context(), id); err != nil {
		h.logger.Error("delete account failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
