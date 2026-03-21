package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreatePaymentParams struct {
	UserID      uuid.UUID
	Provider    string
	ExternalID  string
	AmountCents int32
	Currency    string
	Status      string
	Plan        string
}

func (q *Queries) CreatePayment(ctx context.Context, arg CreatePaymentParams) (Payment, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO payments (user_id, provider, external_id, amount_cents, currency, status, plan)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING *`,
		arg.UserID, arg.Provider, arg.ExternalID, arg.AmountCents, arg.Currency, arg.Status, arg.Plan,
	)
	var p Payment
	err := row.Scan(&p.ID, &p.UserID, &p.Provider, &p.ExternalID, &p.AmountCents, &p.Currency,
		&p.Status, &p.Plan, &p.CurrentPeriodStart, &p.CurrentPeriodEnd, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (q *Queries) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (Payment, error) {
	row := q.db.QueryRow(ctx,
		`SELECT * FROM payments WHERE user_id = $1 AND status = 'active' ORDER BY created_at DESC LIMIT 1`,
		userID)
	var p Payment
	err := row.Scan(&p.ID, &p.UserID, &p.Provider, &p.ExternalID, &p.AmountCents, &p.Currency,
		&p.Status, &p.Plan, &p.CurrentPeriodStart, &p.CurrentPeriodEnd, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

type UpdatePaymentStatusParams struct {
	ID                 uuid.UUID
	Status             string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
}

func (q *Queries) UpdatePaymentStatus(ctx context.Context, arg UpdatePaymentStatusParams) error {
	_, err := q.db.Exec(ctx,
		`UPDATE payments SET status = $2, current_period_start = $3, current_period_end = $4, updated_at = NOW() WHERE id = $1`,
		arg.ID, arg.Status, arg.CurrentPeriodStart, arg.CurrentPeriodEnd)
	return err
}
