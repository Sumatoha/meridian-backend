package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/auth"
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
	userID := auth.UserID(r.Context())

	var req dto.CreateAccountRequest
	if err := parseJSON(r, &req); err != nil {
		h.logger.Warn("accounts.create: failed to parse body",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.IGUsername == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "ig_username is required")
		return
	}

	h.logger.Info("accounts.create: creating account",
		slog.String("ig_username", req.IGUsername),
		slog.String("user_id", userID.String()),
	)

	account, err := h.svc.CreateAccount(r.Context(), req)
	if err != nil {
		h.logger.Error("accounts.create: failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to create account")
		return
	}

	h.logger.Info("accounts.create: success",
		slog.String("account_id", account.ID.String()),
		slog.String("ig_username", req.IGUsername),
	)

	respondJSON(w, http.StatusCreated, account)
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())

	accounts, err := h.svc.ListAccounts(r.Context())
	if err != nil {
		h.logger.Error("accounts.list: failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list accounts")
		return
	}

	h.logger.Info("accounts.list: success",
		slog.String("user_id", userID.String()),
		slog.Int("count", len(accounts)),
	)

	respondJSON(w, http.StatusOK, accounts)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())

	id, ok := parseUUID(chi.URLParam(r, "id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	h.logger.Info("accounts.delete: deleting",
		slog.String("account_id", id.String()),
		slog.String("user_id", userID.String()),
	)

	if err := h.svc.DeleteAccount(r.Context(), id); err != nil {
		h.logger.Error("accounts.delete: failed",
			slog.String("error", err.Error()),
			slog.String("account_id", id.String()),
		)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
