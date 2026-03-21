package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

type AccountService struct {
	db      *pgxpool.Pool
	queries *repository.Queries
}

func NewAccountService(db *pgxpool.Pool, queries *repository.Queries) *AccountService {
	return &AccountService{db: db, queries: queries}
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
