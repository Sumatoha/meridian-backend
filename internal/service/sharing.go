package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

type SharingService struct {
	queries *repository.Queries
	tierSvc *TierService
}

func NewSharingService(queries *repository.Queries, tierSvc *TierService) *SharingService {
	return &SharingService{queries: queries, tierSvc: tierSvc}
}

// CreateShareLink generates a public share token for a plan.
func (s *SharingService) CreateShareLink(ctx context.Context, userID, planID uuid.UUID) (string, error) {
	if err := s.tierSvc.CheckSharing(ctx, userID); err != nil {
		return "", err
	}

	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("create share link: generate token: %w", err)
	}

	_, err = s.queries.SetPlanShareToken(ctx, repository.SetPlanShareTokenParams{
		ID:         planID,
		ShareToken: &token,
	})
	if err != nil {
		return "", fmt.Errorf("create share link: set token: %w", err)
	}

	return token, nil
}

// RevokeShareLink removes the share token from a plan.
func (s *SharingService) RevokeShareLink(ctx context.Context, planID uuid.UUID) error {
	return s.queries.RevokePlanShareToken(ctx, planID)
}

// GetSharedPlan returns a plan by its share token (public, no auth).
func (s *SharingService) GetSharedPlan(ctx context.Context, token string) (dto.ContentPlanDTO, error) {
	plan, err := s.queries.GetPlanByShareToken(ctx, &token)
	if err != nil {
		return dto.ContentPlanDTO{}, fmt.Errorf("shared plan not found")
	}

	slots, err := s.queries.GetSlotsByPlanID(ctx, plan.ID)
	if err != nil {
		return dto.ContentPlanDTO{}, fmt.Errorf("get shared plan slots: %w", err)
	}

	slotDTOs := make([]dto.ContentSlotDTO, 0, len(slots))
	for _, slot := range slots {
		d, err := slotToDTO(slot)
		if err != nil {
			continue
		}
		slotDTOs = append(slotDTOs, d)
	}

	return dto.ContentPlanDTO{
		ContentPlanSummaryDTO: planToSummaryDTO(plan),
		Slots:                 slotDTOs,
	}, nil
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
