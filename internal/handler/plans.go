package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type PlanHandler struct {
	planSvc    *service.PlanService
	accountSvc *service.AccountService
	tierSvc    *service.TierService
	logger     *slog.Logger
}

func NewPlanHandler(planSvc *service.PlanService, accountSvc *service.AccountService, tierSvc *service.TierService, logger *slog.Logger) *PlanHandler {
	return &PlanHandler{planSvc: planSvc, accountSvc: accountSvc, tierSvc: tierSvc, logger: logger}
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
	if err := parseJSON(r, &req); err != nil {
		h.logger.Warn("plan generate: failed to parse body", slog.String("error", err.Error()))
		// Continue with defaults — body is optional
	}

	// Check tier limit synchronously — return 403 immediately if at limit
	userID := auth.UserID(r.Context())
	if err := h.tierSvc.CheckPlanGeneration(r.Context(), userID); err != nil {
		var tierErr *service.TierError
		if errors.As(err, &tierErr) {
			respondJSON(w, http.StatusForbidden, dto.TierLimitError{
				Code:      "tier_limit",
				Message:   tierErr.Message,
				Feature:   tierErr.Feature,
				Limit:     tierErr.Limit,
				Used:      tierErr.Used,
				UpgradeTo: tierErr.UpgradeTo,
			})
			return
		}
		h.logger.Error("tier check failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to check tier limits")
		return
	}

	startDate := time.Now().AddDate(0, 0, 1) // tomorrow
	if req.StartDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.StartDate)
		if err == nil {
			startDate = parsed
		}
	}

	// Background generation with detached context.
	// Returns 202 immediately. Frontend polls GET /plans to check when ready.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		opts := &service.GeneratePlanOptions{}
		if req.ContentLanguage != nil {
			opts.ContentLanguage = *req.ContentLanguage
		}
		if req.PostingFrequency != nil {
			opts.PostingFrequency = *req.PostingFrequency
		}
		if req.ContentGoal != nil {
			opts.ContentGoal = *req.ContentGoal
		}
		opts.MixUseful = req.MixUseful
		opts.MixSelling = req.MixSelling
		opts.MixPersonal = req.MixPersonal
		opts.MixEntertaining = req.MixEntertaining
		if req.BrandContext != nil {
			opts.BrandContext = *req.BrandContext
		}
		planID, err := h.planSvc.GeneratePlan(ctx, userID, accountID, startDate, opts)
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
