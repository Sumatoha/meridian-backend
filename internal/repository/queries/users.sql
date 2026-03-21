-- name: UpsertUser :one
INSERT INTO users (supabase_user_id, email)
VALUES ($1, $2)
ON CONFLICT (supabase_user_id) DO UPDATE SET
    email = EXCLUDED.email,
    updated_at = NOW()
RETURNING *;

-- name: GetUserBySupabaseID :one
SELECT * FROM users WHERE supabase_user_id = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserPlan :one
UPDATE users SET
    plan = $2,
    plan_expires_at = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
