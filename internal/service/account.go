package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/instagram"
	"github.com/meridian/api/internal/repository"
)

type AccountService struct {
	db          *pgxpool.Pool
	queries     *repository.Queries
	oauthClient *instagram.OAuthClient
	appSecret   string
	tierSvc     *TierService
}

func NewAccountService(db *pgxpool.Pool, queries *repository.Queries, oauthClient *instagram.OAuthClient, appSecret string, tierSvc *TierService) *AccountService {
	return &AccountService{db: db, queries: queries, oauthClient: oauthClient, appSecret: appSecret, tierSvc: tierSvc}
}

// EnsureUser creates or updates the internal user record from a Supabase JWT.
func (s *AccountService) EnsureUser(ctx context.Context, supabaseUID uuid.UUID, email string) (uuid.UUID, error) {
	user, err := s.queries.UpsertUser(ctx, repository.UpsertUserParams{
		SupabaseUserID: supabaseUID,
		Email:          email,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("ensure user: %w", err)
	}
	return user.ID, nil
}

// CreateAccount links an Instagram username (manual, no OAuth).
func (s *AccountService) CreateAccount(ctx context.Context, req dto.CreateAccountRequest) (dto.AccountResponse, error) {
	userID := auth.UserID(ctx)

	// Check account limit for user's tier
	if s.tierSvc != nil {
		if err := s.tierSvc.CheckAccountCreation(ctx, userID); err != nil {
			return dto.AccountResponse{}, err
		}
	}

	account, err := s.queries.CreateAccount(ctx, repository.CreateAccountParams{
		UserID:     userID,
		IgUsername: req.IGUsername,
	})
	if err != nil {
		return dto.AccountResponse{}, fmt.Errorf("create account: %w", err)
	}

	return accountToDTO(account), nil
}

// ListAccounts returns all Instagram accounts for the current user.
func (s *AccountService) ListAccounts(ctx context.Context) ([]dto.AccountResponse, error) {
	userID := auth.UserID(ctx)

	accounts, err := s.queries.GetAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	result := make([]dto.AccountResponse, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, accountToDTO(a))
	}
	return result, nil
}

// DeleteAccount removes an Instagram account.
func (s *AccountService) DeleteAccount(ctx context.Context, accountID uuid.UUID) error {
	userID := auth.UserID(ctx)

	if err := s.queries.DeleteAccount(ctx, repository.DeleteAccountParams{
		ID:     accountID,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

// GetAccountForUser fetches and validates account ownership.
func (s *AccountService) GetAccountForUser(ctx context.Context, accountID uuid.UUID) (repository.InstagramAccount, error) {
	userID := auth.UserID(ctx)

	account, err := s.queries.GetAccountByID(ctx, accountID)
	if err != nil {
		return repository.InstagramAccount{}, fmt.Errorf("get account: %w", err)
	}

	if account.UserID != userID {
		return repository.InstagramAccount{}, fmt.Errorf("account not found")
	}

	return account, nil
}

// GetOAuthURL generates the Instagram OAuth authorization URL.
func (s *AccountService) GetOAuthURL(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID) (string, error) {
	if s.oauthClient == nil {
		return "", fmt.Errorf("oauth not configured")
	}

	state, err := instagram.EncodeState(userID, accountID, s.appSecret)
	if err != nil {
		return "", fmt.Errorf("encode oauth state: %w", err)
	}

	return s.oauthClient.BuildAuthURL(state), nil
}

// HandleOAuthCallback exchanges the authorization code for tokens and creates/updates the account.
func (s *AccountService) HandleOAuthCallback(ctx context.Context, code, state string) (dto.OAuthCallbackResponse, error) {
	if s.oauthClient == nil {
		return dto.OAuthCallbackResponse{}, fmt.Errorf("oauth not configured")
	}

	// Verify and decode state
	userID, accountID, err := instagram.DecodeState(state, s.appSecret)
	if err != nil {
		return dto.OAuthCallbackResponse{}, fmt.Errorf("invalid oauth state: %w", err)
	}

	// Exchange code for short-lived token
	shortToken, _, err := s.oauthClient.ExchangeCode(ctx, code)
	if err != nil {
		return dto.OAuthCallbackResponse{}, fmt.Errorf("exchange code: %w", err)
	}

	// Swap for long-lived token (60 days)
	longToken, expiresAt, err := s.oauthClient.ExchangeLongLivedToken(ctx, shortToken)
	if err != nil {
		return dto.OAuthCallbackResponse{}, fmt.Errorf("exchange long-lived token: %w", err)
	}

	// Fetch Instagram profile
	igUserID, username, profilePicURL, err := s.oauthClient.GetProfile(ctx, longToken)
	if err != nil {
		return dto.OAuthCallbackResponse{}, fmt.Errorf("fetch profile: %w", err)
	}

	slog.Info("oauth: profile fetched",
		slog.String("ig_user_id", igUserID),
		slog.String("username", username),
	)

	var account repository.InstagramAccount
	isNew := false

	if accountID != nil {
		// Connect OAuth to existing account
		account, err = s.queries.ConnectOAuthToAccount(ctx, repository.ConnectOAuthToAccountParams{
			ID:             *accountID,
			IgUserID:       &igUserID,
			IgUsername:     username,
			AccessToken:    &longToken,
			TokenExpiresAt: &expiresAt,
			ProfilePicUrl:  &profilePicURL,
		})
		if err != nil {
			return dto.OAuthCallbackResponse{}, fmt.Errorf("connect oauth to account: %w", err)
		}
	} else {
		// Create new OAuth account
		account, err = s.queries.CreateOAuthAccount(ctx, repository.CreateOAuthAccountParams{
			UserID:         userID,
			IgUsername:     username,
			IgUserID:       &igUserID,
			AccessToken:    &longToken,
			TokenExpiresAt: &expiresAt,
		})
		if err != nil {
			return dto.OAuthCallbackResponse{}, fmt.Errorf("create oauth account: %w", err)
		}
		isNew = true
	}

	return dto.OAuthCallbackResponse{
		Account: accountToDTO(account),
		IsNew:   isNew,
	}, nil
}

func accountToDTO(a repository.InstagramAccount) dto.AccountResponse {
	resp := dto.AccountResponse{
		ID:               a.ID,
		IGUsername:        a.IgUsername,
		IsOAuthConnected: a.IsOauthConnected,
	}
	if a.IgUserID != nil {
		resp.IGUserID = a.IgUserID
	}
	if a.ProfilePicUrl != nil {
		resp.ProfilePicURL = a.ProfilePicUrl
	}
	if a.FollowersCount != nil {
		resp.FollowersCount = a.FollowersCount
	}
	return resp
}
