package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

type BillingService struct {
	queries     *repository.Queries
	dodoAPIKey  string
	kaspiID     string
	kaspiSecret string
}

func NewBillingService(queries *repository.Queries, dodoAPIKey, kaspiID, kaspiSecret string) *BillingService {
	return &BillingService{
		queries:     queries,
		dodoAPIKey:  dodoAPIKey,
		kaspiID:     kaspiID,
		kaspiSecret: kaspiSecret,
	}
}

// CreateCheckout generates a checkout URL for the given plan and provider.
func (s *BillingService) CreateCheckout(ctx context.Context, userID uuid.UUID, req dto.CheckoutRequest) (string, error) {
	switch req.Provider {
	case "dodo":
		return s.createDodoCheckout(ctx, userID, req.Plan)
	case "kaspi":
		return s.createKaspiCheckout(ctx, userID, req.Plan)
	default:
		return "", fmt.Errorf("billing: unsupported provider: %s", req.Provider)
	}
}

// GetSubscription returns the active subscription for a user.
func (s *BillingService) GetSubscription(ctx context.Context, userID uuid.UUID) (dto.SubscriptionResponse, error) {
	payment, err := s.queries.GetActiveSubscription(ctx, userID)
	if err != nil {
		return dto.SubscriptionResponse{
			Plan:   "free",
			Status: "active",
		}, nil
	}

	return dto.SubscriptionResponse{
		Plan:      payment.Plan,
		Status:    payment.Status,
		ExpiresAt: payment.CurrentPeriodEnd,
		Provider:  payment.Provider,
	}, nil
}

// HandleDodoWebhook processes a DodoPayments webhook event.
func (s *BillingService) HandleDodoWebhook(ctx context.Context, payload []byte) error {
	// TODO: Verify webhook signature with DodoWebhookSecret
	// TODO: Parse payload, update payment status, update user plan
	_ = payload
	return nil
}

// HandleKaspiWebhook processes a Kaspi Pay webhook event.
func (s *BillingService) HandleKaspiWebhook(ctx context.Context, payload []byte) error {
	// TODO: Verify webhook signature
	// TODO: Parse payload, update payment status, update user plan
	_ = payload
	return nil
}

// ActivateSubscription activates a subscription after successful payment.
func (s *BillingService) ActivateSubscription(ctx context.Context, userID uuid.UUID, provider, externalID, plan string, periodEnd time.Time) error {
	now := time.Now()

	_, err := s.queries.CreatePayment(ctx, repository.CreatePaymentParams{
		UserID:     userID,
		Provider:   provider,
		ExternalID: externalID,
		AmountCents: planPrice(plan),
		Currency:   "USD",
		Status:     "active",
		Plan:       plan,
	})
	if err != nil {
		return fmt.Errorf("billing: create payment: %w", err)
	}

	_, err = s.queries.UpdateUserPlan(ctx, repository.UpdateUserPlanParams{
		ID:            userID,
		Plan:          plan,
		PlanExpiresAt: &periodEnd,
	})
	if err != nil {
		return fmt.Errorf("billing: update user plan: %w", err)
	}

	_ = now
	return nil
}

func (s *BillingService) createDodoCheckout(_ context.Context, _ uuid.UUID, _ string) (string, error) {
	// TODO: Call DodoPayments API to create checkout session
	return "https://checkout.dodopayments.com/placeholder", nil
}

func (s *BillingService) createKaspiCheckout(_ context.Context, _ uuid.UUID, _ string) (string, error) {
	// TODO: Call Kaspi Pay API to create payment link
	return "https://kaspi.kz/pay/placeholder", nil
}

func planPrice(plan string) int32 {
	switch plan {
	case "pro":
		return 1990 // $19.90
	case "agency":
		return 4990 // $49.90
	default:
		return 0
	}
}
