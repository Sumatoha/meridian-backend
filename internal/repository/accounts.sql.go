package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreateAccountParams struct {
	UserID     uuid.UUID
	IgUsername string
}

func (q *Queries) CreateAccount(ctx context.Context, arg CreateAccountParams) (InstagramAccount, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO instagram_accounts (user_id, ig_username, is_oauth_connected)
		VALUES ($1, $2, FALSE)
		RETURNING id, user_id, ig_username, ig_user_id, access_token, token_expires_at,
		profile_pic_url, followers_count, is_oauth_connected, created_at, updated_at`,
		arg.UserID, arg.IgUsername,
	)
	var a InstagramAccount
	err := row.Scan(&a.ID, &a.UserID, &a.IgUsername, &a.IgUserID, &a.AccessToken, &a.TokenExpiresAt,
		&a.ProfilePicUrl, &a.FollowersCount, &a.IsOauthConnected, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

type CreateOAuthAccountParams struct {
	UserID         uuid.UUID
	IgUsername     string
	IgUserID       string
	AccessToken    string
	TokenExpiresAt time.Time
	ProfilePicUrl  *string
}

func (q *Queries) CreateOAuthAccount(ctx context.Context, arg CreateOAuthAccountParams) (InstagramAccount, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO instagram_accounts (user_id, ig_username, ig_user_id, access_token, token_expires_at, profile_pic_url, is_oauth_connected)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		RETURNING id, user_id, ig_username, ig_user_id, access_token, token_expires_at,
		profile_pic_url, followers_count, is_oauth_connected, created_at, updated_at`,
		arg.UserID, arg.IgUsername, arg.IgUserID, arg.AccessToken, arg.TokenExpiresAt, arg.ProfilePicUrl,
	)
	var a InstagramAccount
	err := row.Scan(&a.ID, &a.UserID, &a.IgUsername, &a.IgUserID, &a.AccessToken, &a.TokenExpiresAt,
		&a.ProfilePicUrl, &a.FollowersCount, &a.IsOauthConnected, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

type ConnectOAuthToAccountParams struct {
	ID             uuid.UUID
	IgUserID       string
	IgUsername     string
	AccessToken    string
	TokenExpiresAt time.Time
	ProfilePicUrl  *string
}

func (q *Queries) ConnectOAuthToAccount(ctx context.Context, arg ConnectOAuthToAccountParams) (InstagramAccount, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE instagram_accounts SET
			ig_user_id = $2, ig_username = $3, access_token = $4,
			token_expires_at = $5, profile_pic_url = $6,
			is_oauth_connected = TRUE, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, ig_username, ig_user_id, access_token, token_expires_at,
		profile_pic_url, followers_count, is_oauth_connected, created_at, updated_at`,
		arg.ID, arg.IgUserID, arg.IgUsername, arg.AccessToken, arg.TokenExpiresAt, arg.ProfilePicUrl,
	)
	var a InstagramAccount
	err := row.Scan(&a.ID, &a.UserID, &a.IgUsername, &a.IgUserID, &a.AccessToken, &a.TokenExpiresAt,
		&a.ProfilePicUrl, &a.FollowersCount, &a.IsOauthConnected, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (q *Queries) GetAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]InstagramAccount, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, user_id, ig_username, ig_user_id, access_token, token_expires_at,
		profile_pic_url, followers_count, is_oauth_connected, created_at, updated_at
		FROM instagram_accounts WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []InstagramAccount
	for rows.Next() {
		var a InstagramAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.IgUsername, &a.IgUserID, &a.AccessToken, &a.TokenExpiresAt,
			&a.ProfilePicUrl, &a.FollowersCount, &a.IsOauthConnected, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, nil
}

func (q *Queries) GetAccountByID(ctx context.Context, id uuid.UUID) (InstagramAccount, error) {
	row := q.db.QueryRow(ctx,
		`SELECT id, user_id, ig_username, ig_user_id, access_token, token_expires_at,
		profile_pic_url, followers_count, is_oauth_connected, created_at, updated_at
		FROM instagram_accounts WHERE id = $1`, id)
	var a InstagramAccount
	err := row.Scan(&a.ID, &a.UserID, &a.IgUsername, &a.IgUserID, &a.AccessToken, &a.TokenExpiresAt,
		&a.ProfilePicUrl, &a.FollowersCount, &a.IsOauthConnected, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

type DeleteAccountParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) DeleteAccount(ctx context.Context, arg DeleteAccountParams) error {
	_, err := q.db.Exec(ctx, `DELETE FROM instagram_accounts WHERE id = $1 AND user_id = $2`, arg.ID, arg.UserID)
	return err
}

type UpdateAccountProfileParams struct {
	ID             uuid.UUID
	ProfilePicUrl  *string
	FollowersCount *int32
}

func (q *Queries) UpdateAccountProfile(ctx context.Context, arg UpdateAccountProfileParams) error {
	_, err := q.db.Exec(ctx,
		`UPDATE instagram_accounts SET profile_pic_url = $2, followers_count = $3, updated_at = NOW() WHERE id = $1`,
		arg.ID, arg.ProfilePicUrl, arg.FollowersCount)
	return err
}

type UpdateAccountTokenParams struct {
	ID             uuid.UUID
	AccessToken    *string
	TokenExpiresAt *time.Time
}

func (q *Queries) UpdateAccountToken(ctx context.Context, arg UpdateAccountTokenParams) error {
	_, err := q.db.Exec(ctx,
		`UPDATE instagram_accounts SET access_token = $2, token_expires_at = $3, updated_at = NOW() WHERE id = $1`,
		arg.ID, arg.AccessToken, arg.TokenExpiresAt)
	return err
}

func (q *Queries) GetAccountsWithExpiringTokens(ctx context.Context) ([]InstagramAccount, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, user_id, ig_username, ig_user_id, access_token, token_expires_at,
		profile_pic_url, followers_count, is_oauth_connected, created_at, updated_at
		FROM instagram_accounts
		WHERE is_oauth_connected = TRUE AND token_expires_at < NOW() + INTERVAL '7 days' AND access_token IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []InstagramAccount
	for rows.Next() {
		var a InstagramAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.IgUsername, &a.IgUserID, &a.AccessToken, &a.TokenExpiresAt,
			&a.ProfilePicUrl, &a.FollowersCount, &a.IsOauthConnected, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, nil
}
