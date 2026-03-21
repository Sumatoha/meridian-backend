package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/service"
	"github.com/riverqueue/river"
)

// AnalyzeProfileArgs are the arguments for the profile analysis job.
type AnalyzeProfileArgs struct {
	AccountID uuid.UUID `json:"account_id"`
}

func (AnalyzeProfileArgs) Kind() string { return "analyze_profile" }

// AnalyzeProfileWorker processes profile analysis jobs.
type AnalyzeProfileWorker struct {
	river.WorkerDefaults[AnalyzeProfileArgs]
	analysisSvc *service.AnalysisService
	logger      *slog.Logger
}

func NewAnalyzeProfileWorker(analysisSvc *service.AnalysisService, logger *slog.Logger) *AnalyzeProfileWorker {
	return &AnalyzeProfileWorker{analysisSvc: analysisSvc, logger: logger}
}

func (w *AnalyzeProfileWorker) Work(ctx context.Context, job *river.Job[AnalyzeProfileArgs]) error {
	w.logger.Info("starting profile analysis",
		slog.String("account_id", job.Args.AccountID.String()),
		slog.Int64("job_id", job.ID),
	)

	if err := w.analysisSvc.AnalyzeProfile(ctx, job.Args.AccountID); err != nil {
		return fmt.Errorf("analyze profile job: %w", err)
	}

	w.logger.Info("profile analysis completed",
		slog.String("account_id", job.Args.AccountID.String()),
	)

	return nil
}
