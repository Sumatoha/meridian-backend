package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UpsertUserParams struct {
	SupabaseUserID uuid.UUID
	Email          string
}

func (q *Queries) UpsertUser(ctx context.Context, arg UpsertUserParams) (User, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO users (supabase_user_id, email)
		VALUES ($1, $2)
		ON CONFLICT (supabase_user_id) DO UPDATE SET email = EXCLUDED.email, updated_at = NOW()
		RETURNING id, supabase_user_id, email, plan, plan_expires_at, created_at, updated_at`,
		arg.SupabaseUserID, arg.Email,
	)
	var u User
	err := row.Scan(&u.ID, &u.SupabaseUserID, &u.Email, &u.Plan, &u.PlanExpiresAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (q *Queries) GetUserBySupabaseID(ctx context.Context, supabaseUserID uuid.UUID) (User, error) {
	row := q.db.QueryRow(ctx, `SELECT * FROM users WHERE supabase_user_id = $1`, supabaseUserID)
	var u User
	err := row.Scan(&u.ID, &u.SupabaseUserID, &u.Email, &u.Plan, &u.PlanExpiresAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

type UpdateUserPlanParams struct {
	ID            uuid.UUID
	Plan          string
	PlanExpiresAt *time.Time
}

func (q *Queries) UpdateUserPlan(ctx context.Context, arg UpdateUserPlanParams) (User, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE users SET plan = $2, plan_expires_at = $3, updated_at = NOW() WHERE id = $1
		RETURNING id, supabase_user_id, email, plan, plan_expires_at, created_at, updated_at`,
		arg.ID, arg.Plan, arg.PlanExpiresAt,
	)
	var u User
	err := row.Scan(&u.ID, &u.SupabaseUserID, &u.Email, &u.Plan, &u.PlanExpiresAt, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
