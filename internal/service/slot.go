package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/storage"
)

type SlotService struct {
	queries *repository.Queries
	storage *storage.Client
	tierSvc *TierService
}

func NewSlotService(queries *repository.Queries, storageClient *storage.Client, tierSvc *TierService) *SlotService {
	return &SlotService{queries: queries, storage: storageClient, tierSvc: tierSvc}
}

// GetSlot returns a single content slot.
func (s *SlotService) GetSlot(ctx context.Context, slotID uuid.UUID) (dto.ContentSlotDTO, error) {
	slot, err := s.queries.GetSlotByID(ctx, slotID)
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("get slot: %w", err)
	}

	d, err := slotToDTO(slot)
	if err != nil {
		return dto.ContentSlotDTO{}, err
	}

	// Enrich media with public URLs
	for i := range d.Media {
		d.Media[i].URL = s.storage.GetPublicURL(d.Media[i].StoragePath)
	}

	return d, nil
}

// ListSlots returns all slots for a plan.
func (s *SlotService) ListSlots(ctx context.Context, planID uuid.UUID) ([]dto.ContentSlotDTO, error) {
	slots, err := s.queries.GetSlotsByPlanID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("list slots: %w", err)
	}

	result := make([]dto.ContentSlotDTO, 0, len(slots))
	for _, slot := range slots {
		d, err := slotToDTO(slot)
		if err != nil {
			continue
		}
		for i := range d.Media {
			d.Media[i].URL = s.storage.GetPublicURL(d.Media[i].StoragePath)
		}
		result = append(result, d)
	}
	return result, nil
}

// UpdateSlot updates editable fields of a content slot.
func (s *SlotService) UpdateSlot(ctx context.Context, slotID uuid.UUID, req dto.UpdateSlotRequest) (dto.ContentSlotDTO, error) {
	params := repository.UpdateSlotParams{
		ID:       slotID,
		Caption:  req.Caption,
		Hashtags: req.Hashtags,
		Status:   req.Status,
	}
	if req.IsUserContent != nil {
		params.IsUserContent = pgtype.Bool{Bool: *req.IsUserContent, Valid: true}
	}
	if req.ScheduledTime != nil {
		t, err := time.Parse("15:04", *req.ScheduledTime)
		if err == nil {
			params.ScheduledTime = pgtype.Time{
				Microseconds: int64(t.Hour())*3600000000 + int64(t.Minute())*60000000,
				Valid:        true,
			}
		}
	}
	if req.ScheduledDate != nil {
		d, err := time.Parse("2006-01-02", *req.ScheduledDate)
		if err == nil {
			params.ScheduledDate = pgtype.Date{Time: d, Valid: true}
		}
	}
	slot, err := s.queries.UpdateSlot(ctx, params)
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("update slot: %w", err)
	}

	d, err := slotToDTO(slot)
	if err != nil {
		return dto.ContentSlotDTO{}, err
	}

	for i := range d.Media {
		d.Media[i].URL = s.storage.GetPublicURL(d.Media[i].StoragePath)
	}

	return d, nil
}

// UpdateSlotMedia replaces the media list for a slot.
func (s *SlotService) UpdateSlotMedia(ctx context.Context, slotID uuid.UUID, media []dto.MediaItem) (dto.ContentSlotDTO, error) {
	mediaJSON, err := json.Marshal(media)
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("marshal media: %w", err)
	}

	slot, err := s.queries.UpdateSlotMedia(ctx, repository.UpdateSlotMediaParams{
		ID:    slotID,
		Media: mediaJSON,
	})
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("update slot media: %w", err)
	}

	d, err := slotToDTO(slot)
	if err != nil {
		return dto.ContentSlotDTO{}, err
	}

	for i := range d.Media {
		d.Media[i].URL = s.storage.GetPublicURL(d.Media[i].StoragePath)
	}

	return d, nil
}

// ApproveSlot marks a single slot as approved.
func (s *SlotService) ApproveSlot(ctx context.Context, slotID uuid.UUID) (dto.ContentSlotDTO, error) {
	approved := "approved"
	slot, err := s.queries.UpdateSlot(ctx, repository.UpdateSlotParams{
		ID:     slotID,
		Status: &approved,
	})
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("approve slot: %w", err)
	}

	// Update plan counters
	s.queries.UpdatePlanCounters(ctx, slot.PlanID)

	d, err := slotToDTO(slot)
	if err != nil {
		return dto.ContentSlotDTO{}, err
	}
	return d, nil
}

// MoveSlot changes the scheduled date of a slot.
func (s *SlotService) MoveSlot(ctx context.Context, slotID uuid.UUID, newDate string) (dto.ContentSlotDTO, error) {
	parsed, err := time.Parse("2006-01-02", newDate)
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("move slot: invalid date %q: %w", newDate, err)
	}

	slot, err := s.queries.UpdateSlot(ctx, repository.UpdateSlotParams{
		ID:            slotID,
		ScheduledDate: pgtype.Date{Time: parsed, Valid: true},
	})
	if err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("move slot: %w", err)
	}

	d, err := slotToDTO(slot)
	if err != nil {
		return dto.ContentSlotDTO{}, err
	}
	return d, nil
}

// ApproveAll approves all draft slots (media not required for approval).
func (s *SlotService) ApproveAll(ctx context.Context, planID uuid.UUID) (dto.ApproveAllResponse, error) {
	approvedCount, err := s.queries.ApproveAllDraftSlots(ctx, planID)
	if err != nil {
		return dto.ApproveAllResponse{}, fmt.Errorf("approve all: %w", err)
	}

	// Count how many approved slots still need media (for UI feedback)
	missingMedia, err := s.queries.CountApprovedWithoutMedia(ctx, planID)
	if err != nil {
		return dto.ApproveAllResponse{}, fmt.Errorf("count missing media: %w", err)
	}

	s.queries.UpdatePlanCounters(ctx, planID)

	return dto.ApproveAllResponse{
		ApprovedCount:     int(approvedCount),
		MissingMediaCount: int(missingMedia),
	}, nil
}

// StartPosting queues all approved slots with media for publishing.
// Requires the user's tier to allow auto-posting.
func (s *SlotService) StartPosting(ctx context.Context, userID, planID uuid.UUID) (dto.StartPostingResponse, error) {
	if err := s.tierSvc.CheckAutoPosting(ctx, userID); err != nil {
		return dto.StartPostingResponse{}, err
	}

	queuedCount, err := s.queries.QueueApprovedSlots(ctx, planID)
	if err != nil {
		return dto.StartPostingResponse{}, fmt.Errorf("start posting: %w", err)
	}

	s.queries.UpdatePlanCounters(ctx, planID)

	return dto.StartPostingResponse{
		QueuedCount: int(queuedCount),
	}, nil
}
