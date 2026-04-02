package payments

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// PaymentService defines the business logic operations for Payments.
type PaymentService interface {
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)
	ValidateReference(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error)
	ProcessWebhook(ctx context.Context, payload WebhookPayload) error
}

type paymentService struct {
	repo PaymentRepository
	// In the future, we may inject event publishers or task managers here.
}

// NewPaymentService initializes a new payment service.
func NewPaymentService(repo PaymentRepository) PaymentService {
	return &paymentService{repo: repo}
}

// CreateCheckoutSession saves the initial intent and returns mocked LankaPay session details.
func (s *paymentService) CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	sessionID := "sess_" + fmt.Sprintf("%d", time.Now().UnixNano())
	taskID, ok := req.Metadata["task_id"]
	if !ok {
		return nil, fmt.Errorf("task_id is required in metadata")
	}

	tx := &PaymentTransaction{
		ID:              uuid.NewString(),
		ReferenceNumber: req.ReferenceNumber,
		TaskID:          taskID,
		SessionID:       sessionID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          PaymentStatusPending,
		ExpiryDate:      req.ExpiresAt,
		GatewayMetadata: req.Metadata,
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create payment transaction: %w", err)
	}

	slog.Info("created checkout session", "reference_number", req.ReferenceNumber, "session_id", sessionID)

	return &CreateCheckoutResponse{
		SessionID:   sessionID,
		CheckoutURL: "https://sandbox.govpay.lk/checkout/" + sessionID,
		ExpiresIn:   int(time.Until(req.ExpiresAt).Seconds()),
	}, nil
}

// ValidateReference is called by GovPay when a user searches for their reference number.
func (s *paymentService) ValidateReference(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
	slog.Info("validating incoming payment reference", "reference", req.PaymentReference)

	tx, err := s.repo.GetByReferenceNumber(ctx, req.PaymentReference)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment reference: %w", err)
	}
	if tx == nil {
		return &ValidateReferenceResponse{IsPayable: false, Remarks: "Invalid reference number"}, nil
	}

	isPayable := tx.Status == PaymentStatusPending && time.Now().Before(tx.ExpiryDate)

	return &ValidateReferenceResponse{
		Amount:     tx.Amount,
		Currency:   tx.Currency,
		TraderName: "Sample Trader", // TODO: Fetch from actual domain models/context
		OGAName:    "Sample OGA",    // TODO: Fetch from actual domain models/context
		ExpiryDate: tx.ExpiryDate.Format(time.RFC3339),
		IsPayable:  isPayable,
		Remarks:    fmt.Sprintf("Current status: %s", tx.Status),
	}, nil
}

// ProcessWebhook processes asynchronous success/failure updates from GovPay.
func (s *paymentService) ProcessWebhook(ctx context.Context, payload WebhookPayload) error {
	slog.Info("processing payment webhook", "reference_number", payload.ReferenceNumber, "status", payload.Status)

	tx, err := s.repo.GetByReferenceNumber(ctx, payload.ReferenceNumber)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment by reference: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("payment reference not found: %s", payload.ReferenceNumber)
	}

	// Idempotency: Ignore if we already recorded a final status
	if tx.Status == payload.Status || tx.Status == PaymentStatusSuccess {
		slog.Info("webhook ignored (idempotent)", "reference", tx.ReferenceNumber, "current_status", tx.Status)
		return nil
	}

	tx.Status = payload.Status
	tx.PaymentMethod = payload.PaymentMethod

	if tx.GatewayMetadata == nil {
		tx.GatewayMetadata = make(map[string]string)
	}
	tx.GatewayMetadata["gateway_transaction_id"] = payload.GatewayTransactionID
	tx.GatewayMetadata["webhook_timestamp"] = payload.Timestamp

	if err := s.repo.Update(ctx, tx); err != nil {
		return fmt.Errorf("failed to update payment transaction status: %w", err)
	}

	slog.Info("payment transaction updated successfully", "reference", tx.ReferenceNumber, "status", tx.Status)

	// In a complete implementation, we'd emit an internal event here:
	// "PaymentConfirmedEvent" for the Task Engine to pick up.
	// We leave this uncoupled as requested ("ONLY focus first on implementing Payment Service").

	return nil
}
