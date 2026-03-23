package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/tier"
)

// TierError is returned when a tier limit is exceeded.
type TierError struct {
	Feature   string
	Limit     int
	Used      int
	UpgradeTo string
	Message   string
}

func (e *TierError) Error() string {
	return e.Message
}

type TierService struct {
	queries *repository.Queries
}

func NewTierService(queries *repository.Queries) *TierService {
	return &TierService{queries: queries}
}

// CheckPlanGeneration verifies the user hasn't exceeded their monthly plan generation limit.
func (s *TierService) CheckPlanGeneration(ctx context.Context, userID uuid.UUID) error {
	cfg, err := s.getUserTierConfig(ctx, userID)
	if err != nil {
		return fmt.Errorf("check plan generation: %w", err)
	}

	// Unlimited plans (-1) always pass
	if cfg.PlanGenerations < 0 {
		return nil
	}

	used, err := s.queries.GetMonthlyUsage(ctx, repository.GetMonthlyUsageParams{
		UserID: userID,
		Action: "plan_generation",
	})
	if err != nil {
		return fmt.Errorf("check plan generation: get usage: %w", err)
	}

	if int(used) >= cfg.PlanGenerations {
		upgradeTo := "pro"
		if cfg.Name == tier.Pro {
			upgradeTo = "business"
		}
		return &TierError{
			Feature:   "plan_generation",
			Limit:     cfg.PlanGenerations,
			Used:      int(used),
			UpgradeTo: upgradeTo,
			Message:   fmt.Sprintf("Plan generation limit reached (%d/%d this month). Upgrade to %s for more.", used, cfg.PlanGenerations, upgradeTo),
		}
	}

	return nil
}

// CheckAccountCreation verifies the user hasn't exceeded their account limit.
func (s *TierService) CheckAccountCreation(ctx context.Context, userID uuid.UUID) error {
	cfg, err := s.getUserTierConfig(ctx, userID)
	if err != nil {
		return fmt.Errorf("check account creation: %w", err)
	}

	count, err := s.queries.CountAccountsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("check account creation: count: %w", err)
	}

	if int(count) >= cfg.MaxAccounts {
		upgradeTo := "business"
		if cfg.Name == tier.Business {
			return &TierError{
				Feature: "account_creation",
				Limit:   cfg.MaxAccounts,
				Used:    int(count),
				Message: fmt.Sprintf("Account limit reached (%d/%d).", count, cfg.MaxAccounts),
			}
		}
		return &TierError{
			Feature:   "account_creation",
			Limit:     cfg.MaxAccounts,
			Used:      int(count),
			UpgradeTo: upgradeTo,
			Message:   fmt.Sprintf("Account limit reached (%d/%d). Upgrade to %s for more.", count, cfg.MaxAccounts, upgradeTo),
		}
	}

	return nil
}

// CheckAutoPosting verifies the user's tier allows auto-posting.
func (s *TierService) CheckAutoPosting(ctx context.Context, userID uuid.UUID) error {
	cfg, err := s.getUserTierConfig(ctx, userID)
	if err != nil {
		return fmt.Errorf("check auto posting: %w", err)
	}

	if !cfg.AutoPosting {
		return &TierError{
			Feature:   "auto_posting",
			UpgradeTo: "pro",
			Message:   "Auto-posting is available on Pro and Business plans.",
		}
	}

	return nil
}

// CheckExport verifies the user's tier allows export.
func (s *TierService) CheckExport(ctx context.Context, userID uuid.UUID) error {
	cfg, err := s.getUserTierConfig(ctx, userID)
	if err != nil {
		return fmt.Errorf("check export: %w", err)
	}

	if !cfg.Export {
		return &TierError{
			Feature:   "export",
			UpgradeTo: "pro",
			Message:   "Export is available on Pro and Business plans.",
		}
	}

	return nil
}

// CheckSharing verifies the user's tier allows plan sharing.
func (s *TierService) CheckSharing(ctx context.Context, userID uuid.UUID) error {
	cfg, err := s.getUserTierConfig(ctx, userID)
	if err != nil {
		return fmt.Errorf("check sharing: %w", err)
	}

	if !cfg.Sharing {
		return &TierError{
			Feature:   "sharing",
			UpgradeTo: "business",
			Message:   "Plan sharing is available on the Business plan.",
		}
	}

	return nil
}

// GetTierInfo returns the user's current tier, limits, and usage for the frontend.
func (s *TierService) GetTierInfo(ctx context.Context, userID uuid.UUID) (dto.TierInfoResponse, error) {
	plan, err := s.queries.GetUserPlan(ctx, userID)
	if err != nil {
		return dto.TierInfoResponse{}, fmt.Errorf("get tier info: %w", err)
	}

	cfg := tier.Get(plan)

	accountCount, err := s.queries.CountAccountsByUserID(ctx, userID)
	if err != nil {
		return dto.TierInfoResponse{}, fmt.Errorf("get tier info: count accounts: %w", err)
	}

	planGenUsed, err := s.queries.GetMonthlyUsage(ctx, repository.GetMonthlyUsageParams{
		UserID: userID,
		Action: "plan_generation",
	})
	if err != nil {
		return dto.TierInfoResponse{}, fmt.Errorf("get tier info: get usage: %w", err)
	}

	// Calculate next month reset date
	now := time.Now()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	return dto.TierInfoResponse{
		Plan:                 plan,
		MaxAccounts:          cfg.MaxAccounts,
		AccountsUsed:         int(accountCount),
		PlanGenerationsLimit: cfg.PlanGenerations,
		PlanGenerationsUsed:  int(planGenUsed),
		AutoPosting:          cfg.AutoPosting,
		Export:               cfg.Export,
		Sharing:              cfg.Sharing,
		PeriodResetsAt:       nextMonth.Format("2006-01-02"),
	}, nil
}

// IncrementPlanGeneration bumps the monthly plan generation counter.
func (s *TierService) IncrementPlanGeneration(ctx context.Context, userID uuid.UUID) error {
	return s.queries.IncrementUsage(ctx, repository.IncrementUsageParams{
		UserID: userID,
		Action: "plan_generation",
	})
}

func (s *TierService) getUserTierConfig(ctx context.Context, userID uuid.UUID) (tier.Config, error) {
	plan, err := s.queries.GetUserPlan(ctx, userID)
	if err != nil {
		return tier.Config{}, fmt.Errorf("get user plan: %w", err)
	}
	return tier.Get(plan), nil
}
