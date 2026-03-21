package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/instagram"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/storage"
)

type PublisherService struct {
	queries   *repository.Queries
	publisher *instagram.Publisher
	storage   *storage.Client
	logger    *slog.Logger
}

func NewPublisherService(
	queries *repository.Queries,
	publisher *instagram.Publisher,
	storageClient *storage.Client,
	logger *slog.Logger,
) *PublisherService {
	return &PublisherService{
		queries:   queries,
		publisher: publisher,
		storage:   storageClient,
		logger:    logger,
	}
}

// PublishSlotByRecord publishes using already-loaded records.
func (ps *PublisherService) PublishSlotByRecord(ctx context.Context, slot repository.ContentSlot, account repository.InstagramAccount) error {
	return ps.publishSlotInternal(ctx, slot, account)
}

func (ps *PublisherService) publishSlotInternal(ctx context.Context, slot repository.ContentSlot, account repository.InstagramAccount) error {
	if account.IgUserID == nil || account.AccessToken == nil {
		return fmt.Errorf("publish: account not OAuth connected")
	}

	// Parse media
	var media []dto.MediaItem
	if err := json.Unmarshal(slot.Media, &media); err != nil || len(media) == 0 {
		return fmt.Errorf("publish: no media available")
	}

	// Build caption with hashtags
	caption := slot.Caption
	if len(slot.Hashtags) > 0 {
		caption += "\n\n"
		for _, tag := range slot.Hashtags {
			caption += "#" + tag + " "
		}
	}

	// Get public URLs for media
	var mediaURLs []string
	for _, m := range media {
		mediaURLs = append(mediaURLs, ps.storage.GetPublicURL(m.StoragePath))
	}

	// Update status to publishing
	ps.queries.UpdateSlotStatus(ctx, repository.UpdateSlotStatusParams{
		ID:     slot.ID,
		Status: "publishing",
	})

	var result instagram.PublishResult
	var publishErr error

	switch slot.Format {
	case "photo":
		result, publishErr = ps.publisher.PublishPhoto(ctx, *account.IgUserID, *account.AccessToken, mediaURLs[0], caption)
	case "carousel":
		result, publishErr = ps.publisher.PublishCarousel(ctx, *account.IgUserID, *account.AccessToken, mediaURLs, caption)
	case "reels":
		result, publishErr = ps.publisher.PublishReels(ctx, *account.IgUserID, *account.AccessToken, mediaURLs[0], caption)
	default:
		publishErr = fmt.Errorf("unknown format: %s", slot.Format)
	}

	if publishErr != nil {
		errMsg := publishErr.Error()
		ps.queries.UpdateSlotStatus(ctx, repository.UpdateSlotStatusParams{
			ID:           slot.ID,
			Status:       "failed",
			ErrorMessage: &errMsg,
		})
		ps.queries.IncrementSlotRetry(ctx, slot.ID)
		return fmt.Errorf("publish: %w", publishErr)
	}

	// Success
	permalink := result.Permalink
	ps.queries.UpdateSlotPublished(ctx, repository.UpdateSlotPublishedParams{
		ID:        slot.ID,
		IgPostID:  &result.IGPostID,
		IgPostUrl: &permalink,
	})

	// Update plan counters
	ps.queries.UpdatePlanCounters(ctx, slot.PlanID)

	ps.logger.Info("slot published",
		slog.String("slot_id", slot.ID.String()),
		slog.String("ig_post_id", result.IGPostID),
	)

	return nil
}
