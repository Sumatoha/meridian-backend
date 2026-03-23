package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type PlanHandler struct {
	planSvc    *service.PlanService
	accountSvc *service.AccountService
	logger     *slog.Logger
}

func NewPlanHandler(planSvc *service.PlanService, accountSvc *service.AccountService, logger *slog.Logger) *PlanHandler {
	return &PlanHandler{planSvc: planSvc, accountSvc: accountSvc, logger: logger}
}

func (h *PlanHandler) Generate(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseUUID(chi.URLParam(r, "account_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	if _, err := h.accountSvc.GetAccountForUser(r.Context(), accountID); err != nil {
		respondError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}

	var req dto.GeneratePlanRequest
	parseJSON(r, &req)

	startDate := time.Now().AddDate(0, 0, 1) // tomorrow
	if req.StartDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.StartDate)
		if err == nil {
			startDate = parsed
		}
	}

	// Background generation with detached context.
	// Returns 202 immediately. Frontend polls GET /plans list until new plan appears with slots.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		var opts *service.GeneratePlanOptions
		if req.ContentLanguage != nil {
			opts = &service.GeneratePlanOptions{ContentLanguage: *req.ContentLanguage}
		}
		planID, err := h.planSvc.GeneratePlan(ctx, accountID, startDate, opts)
		if err != nil {
			h.logger.Error("plan generation failed",
				slog.String("account_id", accountID.String()),
				slog.String("error", err.Error()),
			)
			return
		}
		h.logger.Info("plan generation completed",
			slog.String("plan_id", planID.String()),
			slog.String("account_id", accountID.String()),
		)
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "generating",
		"message": "Plan generation started. Poll GET /plans to check when ready.",
	})
}

func (h *PlanHandler) List(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseUUID(chi.URLParam(r, "account_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	plans, err := h.planSvc.ListPlans(r.Context(), accountID)
	if err != nil {
		h.logger.Error("list plans failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list plans")
		return
	}

	respondJSON(w, http.StatusOK, plans)
}

func (h *PlanHandler) Get(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	plan, err := h.planSvc.GetPlan(r.Context(), planID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "plan not found")
		return
	}

	respondJSON(w, http.StatusOK, plan)
}

func (h *PlanHandler) Update(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	var req dto.UpdatePlanRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if err := h.planSvc.UpdatePlanStatus(r.Context(), planID, req.Status); err != nil {
		h.logger.Error("update plan failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update plan")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PlanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	if err := h.planSvc.DeletePlan(r.Context(), planID); err != nil {
		h.logger.Error("delete plan failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to delete plan")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
