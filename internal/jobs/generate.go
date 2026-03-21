package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/service"
	"github.com/riverqueue/river"
)

// GeneratePlanArgs are the arguments for the plan generation job.
type GeneratePlanArgs struct {
	AccountID uuid.UUID `json:"account_id"`
	StartDate string    `json:"start_date"` // "2006-01-02" format
}

func (GeneratePlanArgs) Kind() string { return "generate_plan" }

// GeneratePlanWorker processes plan generation jobs.
type GeneratePlanWorker struct {
	river.WorkerDefaults[GeneratePlanArgs]
	planSvc *service.PlanService
	logger  *slog.Logger
}

func NewGeneratePlanWorker(planSvc *service.PlanService, logger *slog.Logger) *GeneratePlanWorker {
	return &GeneratePlanWorker{planSvc: planSvc, logger: logger}
}

func (w *GeneratePlanWorker) Work(ctx context.Context, job *river.Job[GeneratePlanArgs]) error {
	w.logger.Info("starting plan generation",
		slog.String("account_id", job.Args.AccountID.String()),
		slog.Int64("job_id", job.ID),
	)

	startDate, err := time.Parse("2006-01-02", job.Args.StartDate)
	if err != nil {
		return fmt.Errorf("generate plan job: parse start date: %w", err)
	}

	planID, err := w.planSvc.GeneratePlan(ctx, job.Args.AccountID, startDate)
	if err != nil {
		return fmt.Errorf("generate plan job: %w", err)
	}

	w.logger.Info("plan generation completed",
		slog.String("plan_id", planID.String()),
	)

	return nil
}
