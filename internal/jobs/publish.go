package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/service"
	"github.com/riverqueue/river"
)

// PublishSlotArgs are the arguments for the slot publishing job.
type PublishSlotArgs struct {
	SlotID    uuid.UUID `json:"slot_id"`
	AccountID uuid.UUID `json:"account_id"`
}

func (PublishSlotArgs) Kind() string { return "publish_slot" }

// PublishSlotWorker processes slot publishing jobs.
type PublishSlotWorker struct {
	river.WorkerDefaults[PublishSlotArgs]
	publisherSvc *service.PublisherService
	queries      *repository.Queries
	logger       *slog.Logger
}

func NewPublishSlotWorker(publisherSvc *service.PublisherService, queries *repository.Queries, logger *slog.Logger) *PublishSlotWorker {
	return &PublishSlotWorker{publisherSvc: publisherSvc, queries: queries, logger: logger}
}

func (w *PublishSlotWorker) Work(ctx context.Context, job *river.Job[PublishSlotArgs]) error {
	w.logger.Info("publishing slot",
		slog.String("slot_id", job.Args.SlotID.String()),
		slog.Int64("job_id", job.ID),
	)

	slot, err := w.queries.GetSlotByID(ctx, job.Args.SlotID)
	if err != nil {
		return fmt.Errorf("publish job: get slot: %w", err)
	}

	account, err := w.queries.GetAccountByID(ctx, job.Args.AccountID)
	if err != nil {
		return fmt.Errorf("publish job: get account: %w", err)
	}

	if err := w.publisherSvc.PublishSlotByRecord(ctx, slot, account); err != nil {
		// Check retry count
		if slot.RetryCount >= 2 {
			w.logger.Error("slot publishing permanently failed after 3 attempts",
				slog.String("slot_id", slot.ID.String()),
			)
			return nil // Don't retry further
		}
		return fmt.Errorf("publish job: %w", err)
	}

	return nil
}
