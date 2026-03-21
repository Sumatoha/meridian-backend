package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreatePlanParams struct {
	InstagramAccountID uuid.UUID
	Title              string
	StartDate          time.Time
	EndDate            time.Time
	TotalSlots         int32
}

func (q *Queries) CreatePlan(ctx context.Context, arg CreatePlanParams) (ContentPlan, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO content_plans (instagram_account_id, title, start_date, end_date, total_slots)
		VALUES ($1,$2,$3,$4,$5) RETURNING *`,
		arg.InstagramAccountID, arg.Title, arg.StartDate, arg.EndDate, arg.TotalSlots,
	)
	var p ContentPlan
	err := row.Scan(&p.ID, &p.InstagramAccountID, &p.Title, &p.StartDate, &p.EndDate, &p.Status,
		&p.TotalSlots, &p.ApprovedSlots, &p.PublishedSlots, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (q *Queries) GetPlansByAccountID(ctx context.Context, accountID uuid.UUID) ([]ContentPlan, error) {
	rows, err := q.db.Query(ctx,
		`SELECT * FROM content_plans WHERE instagram_account_id = $1 ORDER BY created_at DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ContentPlan
	for rows.Next() {
		var p ContentPlan
		if err := rows.Scan(&p.ID, &p.InstagramAccountID, &p.Title, &p.StartDate, &p.EndDate, &p.Status,
			&p.TotalSlots, &p.ApprovedSlots, &p.PublishedSlots, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, nil
}

func (q *Queries) GetPlanByID(ctx context.Context, id uuid.UUID) (ContentPlan, error) {
	row := q.db.QueryRow(ctx, `SELECT * FROM content_plans WHERE id = $1`, id)
	var p ContentPlan
	err := row.Scan(&p.ID, &p.InstagramAccountID, &p.Title, &p.StartDate, &p.EndDate, &p.Status,
		&p.TotalSlots, &p.ApprovedSlots, &p.PublishedSlots, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

type UpdatePlanStatusParams struct {
	ID     uuid.UUID
	Status string
}

func (q *Queries) UpdatePlanStatus(ctx context.Context, arg UpdatePlanStatusParams) (ContentPlan, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE content_plans SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING *`,
		arg.ID, arg.Status)
	var p ContentPlan
	err := row.Scan(&p.ID, &p.InstagramAccountID, &p.Title, &p.StartDate, &p.EndDate, &p.Status,
		&p.TotalSlots, &p.ApprovedSlots, &p.PublishedSlots, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (q *Queries) UpdatePlanCounters(ctx context.Context, planID uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`UPDATE content_plans SET
		approved_slots = (SELECT COUNT(*) FROM content_slots WHERE plan_id = $1 AND status IN ('approved','queued','publishing','published')),
		published_slots = (SELECT COUNT(*) FROM content_slots WHERE plan_id = $1 AND status = 'published'),
		updated_at = NOW() WHERE id = $1`, planID)
	return err
}

func (q *Queries) DeletePlan(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, `DELETE FROM content_plans WHERE id = $1`, id)
	return err
}
