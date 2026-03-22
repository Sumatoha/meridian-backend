package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type AnalysisHandler struct {
	analysisSvc *service.AnalysisService
	accountSvc  *service.AccountService
	logger      *slog.Logger
}

func NewAnalysisHandler(analysisSvc *service.AnalysisService, accountSvc *service.AccountService, logger *slog.Logger) *AnalysisHandler {
	return &AnalysisHandler{analysisSvc: analysisSvc, accountSvc: accountSvc, logger: logger}
}

// Analyze triggers async profile analysis.
// In production this would enqueue a River job; for now it runs synchronously.
func (h *AnalysisHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseUUID(chi.URLParam(r, "account_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	// Verify ownership
	if _, err := h.accountSvc.GetAccountForUser(r.Context(), accountID); err != nil {
		respondError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}

	// Run analysis in background — use detached context since HTTP response is sent immediately
	go func() {
		ctx := context.Background()
		if err := h.analysisSvc.AnalyzeProfile(ctx, accountID); err != nil {
			h.logger.Error("analysis failed", slog.String("account_id", accountID.String()), slog.String("error", err.Error()))
		}
	}()

	respondJSON(w, http.StatusAccepted, dto.JobResponse{JobID: accountID})
}

func (h *AnalysisHandler) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseUUID(chi.URLParam(r, "account_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	result, err := h.analysisSvc.GetLatestAnalysis(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "no analysis found")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
