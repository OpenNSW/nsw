package paymentsv2

import (
	"context"
	"fmt"
	"log/slog"
)

// PaymentService defines the high-level orchestration for payments.
type PaymentService interface {
	// ListAvailableMethods returns the rendering information for all active payment gateways.
	ListAvailableMethods(ctx context.Context) ([]PaymentProviderInfo, error)

	// CreateCheckoutSession initializes a payment session and generates a ReferenceNumber.
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)

	// ValidateReference is used for real-time validation requests from gateways.
	ValidateReference(ctx context.Context, providerID string, req ValidateReferenceRequest) (*ValidateReferenceResponse, error)

	// ProcessWebhook handles asynchronous notifications from payment gateways.
	ProcessWebhook(ctx context.Context, providerID string, body []byte, headers map[string][]string) error
}

type paymentService struct {
	repo     PaymentRepository
	registry PaymentRegistry
}

// NewPaymentService initializes a new payment service.
func NewPaymentService(repo PaymentRepository, registry PaymentRegistry) PaymentService {
	return &paymentService{
		repo:     repo,
		registry: registry,
	}
}

func (s *paymentService) ListAvailableMethods(ctx context.Context) ([]PaymentProviderInfo, error) {
	return s.registry.ListInfo(), nil
}

func (s *paymentService) CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	// TODO: Implement multi-provider orchestration logic:
	// 1. Select provider from registry
	// 2. Generate a unique NSW ReferenceNumber (e.g., NSW-PR-YYYY-XXXXX)
	// 3. Call provider.CreateSession(ctx, req, generatedRef)
	// 4. Persist transaction via repo including the generatedRef
	// 5. Return CreateCheckoutResponse containing the generatedRef
	return nil, fmt.Errorf("multi-provider checkout orchestration not yet implemented")
}

func (s *paymentService) ValidateReference(ctx context.Context, providerID string, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
	slog.Info("validating incoming payment reference", "provider", providerID, "reference", req.PaymentReference)

	// 1. Get the provider from the registry using the ID from the URL
	provider, err := s.registry.Get(providerID)
	if err != nil {
		return nil, fmt.Errorf("provider %s not found: %w", providerID, err)
	}

	// 2. Look up the transaction metadata from the DB
	tx, err := s.repo.GetByReferenceNumber(ctx, req.PaymentReference)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment reference: %w", err)
	}
	if tx == nil {
		return &ValidateReferenceResponse{IsPayable: false, Remarks: "Invalid reference number"}, nil
	}

	// 3. Security/Consistency check: ensure the transaction was actually intended for this provider
	if tx.ProviderID != providerID {
		slog.Warn("provider mismatch during validation", "expected", tx.ProviderID, "received", providerID, "reference", tx.ReferenceNumber)
		return &ValidateReferenceResponse{IsPayable: false, Remarks: "Reference mismatch"}, nil
	}

	// 4. Delegate final validation response to the provider, injecting the transaction metadata
	return provider.HandleValidateReference(ctx, tx)
}

func (s *paymentService) ProcessWebhook(ctx context.Context, providerID string, body []byte, headers map[string][]string) error {
	// TODO: Implement provider-based webhook processing
	return fmt.Errorf("webhook processing not yet implemented for provider: %s", providerID)
}
