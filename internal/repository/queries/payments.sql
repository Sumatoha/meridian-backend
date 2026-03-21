-- name: CreatePayment :one
INSERT INTO payments (user_id, provider, external_id, amount_cents, currency, status, plan)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPaymentByExternalID :one
SELECT * FROM payments WHERE provider = $1 AND external_id = $2;

-- name: UpdatePaymentStatus :exec
UPDATE payments SET
    status = $2,
    current_period_start = $3,
    current_period_end = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: GetActiveSubscription :one
SELECT * FROM payments
WHERE user_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT 1;
