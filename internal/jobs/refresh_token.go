package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/meridian/api/internal/instagram"
	"github.com/meridian/api/internal/repository"
	"github.com/riverqueue/river"
)

// RefreshTokenArgs are the arguments for the token refresh job.
type RefreshTokenArgs struct{}

func (RefreshTokenArgs) Kind() string { return "refresh_tokens" }

// RefreshTokenWorker refreshes expiring Meta long-lived tokens.
type RefreshTokenWorker struct {
	river.WorkerDefaults[RefreshTokenArgs]
	queries   *repository.Queries
	publisher *instagram.Publisher
	logger    *slog.Logger
}

func NewRefreshTokenWorker(queries *repository.Queries, publisher *instagram.Publisher, logger *slog.Logger) *RefreshTokenWorker {
	return &RefreshTokenWorker{queries: queries, publisher: publisher, logger: logger}
}

func (w *RefreshTokenWorker) Work(ctx context.Context, job *river.Job[RefreshTokenArgs]) error {
	w.logger.Info("starting token refresh check")

	accounts, err := w.queries.GetAccountsWithExpiringTokens(ctx)
	if err != nil {
		return fmt.Errorf("refresh tokens: get accounts: %w", err)
	}

	w.logger.Info("found accounts with expiring tokens", slog.Int("count", len(accounts)))

	for _, account := range accounts {
		if account.AccessToken == nil {
			continue
		}

		newToken, expiresAt, err := w.publisher.RefreshLongLivedToken(ctx, *account.AccessToken)
		if err != nil {
			w.logger.Error("token refresh failed",
				slog.String("account_id", account.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		if err := w.queries.UpdateAccountToken(ctx, repository.UpdateAccountTokenParams{
			ID:             account.ID,
			AccessToken:    &newToken,
			TokenExpiresAt: &expiresAt,
		}); err != nil {
			w.logger.Error("save refreshed token failed",
				slog.String("account_id", account.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		w.logger.Info("token refreshed",
			slog.String("account_id", account.ID.String()),
			slog.Time("new_expiry", expiresAt),
		)
	}

	return nil
}
