-- name: CreatePlan :one
INSERT INTO content_plans (instagram_account_id, title, start_date, end_date, total_slots)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPlansByAccountID :many
SELECT * FROM content_plans
WHERE instagram_account_id = $1
ORDER BY created_at DESC;

-- name: GetPlanByID :one
SELECT * FROM content_plans WHERE id = $1;

-- name: UpdatePlanStatus :one
UPDATE content_plans SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdatePlanCounters :exec
UPDATE content_plans SET
    approved_slots = (SELECT COUNT(*) FROM content_slots WHERE plan_id = $1 AND status IN ('approved', 'queued', 'publishing', 'published')),
    published_slots = (SELECT COUNT(*) FROM content_slots WHERE plan_id = $1 AND status = 'published'),
    updated_at = NOW()
WHERE id = $1;

-- name: DeletePlan :exec
DELETE FROM content_plans WHERE id = $1;
