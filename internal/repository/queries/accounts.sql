-- name: CreateAccount :one
INSERT INTO instagram_accounts (user_id, ig_username, is_oauth_connected)
VALUES ($1, $2, FALSE)
RETURNING *;

-- name: CreateOAuthAccount :one
INSERT INTO instagram_accounts (user_id, ig_username, ig_user_id, access_token, token_expires_at, is_oauth_connected)
VALUES ($1, $2, $3, $4, $5, TRUE)
RETURNING *;

-- name: GetAccountsByUserID :many
SELECT * FROM instagram_accounts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetAccountByID :one
SELECT * FROM instagram_accounts WHERE id = $1;

-- name: DeleteAccount :exec
DELETE FROM instagram_accounts WHERE id = $1 AND user_id = $2;

-- name: UpdateAccountProfile :exec
UPDATE instagram_accounts SET
    profile_pic_url = $2,
    followers_count = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateAccountToken :exec
UPDATE instagram_accounts SET
    access_token = $2,
    token_expires_at = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ConnectOAuthToAccount :one
UPDATE instagram_accounts SET
    ig_user_id = $2,
    ig_username = $3,
    access_token = $4,
    token_expires_at = $5,
    profile_pic_url = $6,
    is_oauth_connected = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetAccountsWithExpiringTokens :many
SELECT * FROM instagram_accounts
WHERE is_oauth_connected = TRUE
  AND token_expires_at < NOW() + INTERVAL '7 days'
  AND access_token IS NOT NULL;
