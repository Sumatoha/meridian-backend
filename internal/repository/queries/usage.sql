-- name: GetMonthlyUsage :one
SELECT COALESCE(
    (SELECT used_count FROM usage_tracking
     WHERE user_id = $1
       AND action = $2
       AND period_start = date_trunc('month', CURRENT_DATE)::date),
    0
)::int AS used_count;

-- name: IncrementUsage :exec
INSERT INTO usage_tracking (user_id, action, period_start, period_end, used_count)
VALUES (
    $1, $2,
    date_trunc('month', CURRENT_DATE)::date,
    (date_trunc('month', CURRENT_DATE) + INTERVAL '1 month' - INTERVAL '1 day')::date,
    1
)
ON CONFLICT (user_id, action, period_start)
DO UPDATE SET
    used_count = usage_tracking.used_count + 1,
    updated_at = NOW();

-- name: CountAccountsByUserID :one
SELECT COUNT(*)::int FROM instagram_accounts WHERE user_id = $1;

-- name: DeletePlansByAccountID :exec
DELETE FROM content_plans WHERE instagram_account_id = $1;

-- name: GetActivePlanByAccountID :one
SELECT * FROM content_plans
WHERE instagram_account_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetUserPlan :one
SELECT plan FROM users WHERE id = $1;
