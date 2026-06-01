package paymentsv2

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/OpenNSW/nsw/backend/internal/paymentsv2/gateways"
	"github.com/google/uuid"
)

// PaymentService defines the high-level orchestration for payments.
type PaymentService interface {
	// ListAvailableMethods returns the rendering information for all active payment gateways.
	ListAvailableMethods(ctx context.Context) ([]GatewayInfo, error)

	// CreateCheckoutSession initializes a payment session and generates a ReferenceNumber.
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)

	// ValidateReference is used for real-time validation requests from gateways.
	ValidateReference(ctx context.Context, gatewayID string, rawBody json.RawMessage) (*gateways.ValidationResponse, error)

	// ProcessWebhook handles asynchronous notifications from payment gateways.
	ProcessWebhook(ctx context.Context, gatewayID string, body []byte, headers map[string][]string) error
}

type paymentService struct {
	repo     PaymentRepository
	registry GatewayRegistry
}

// NewPaymentService initializes a new payment service.
func NewPaymentService(repo PaymentRepository, registry GatewayRegistry) PaymentService {
	return &paymentService{
		repo:     repo,
		registry: registry,
	}
}

func (s *paymentService) ListAvailableMethods(ctx context.Context) ([]GatewayInfo, error) {
	return s.registry.ListInfo(), nil
}

func (s *paymentService) CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	gateway, err := s.registry.Get(req.GatewayID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway %s: %w", req.GatewayID, err)
	}

	taskID, ok := req.Metadata["task_id"]
	if !ok {
		return nil, fmt.Errorf("task_id is required in metadata")
	}

	// 1. Generate a unique NSW ReferenceNumber (e.g., NSW-PR-YYYYMMDD-XXXXX)
	generatedRef := fmt.Sprintf("NSW-PR-%s-%s", time.Now().Format("20060102"), uuid.NewString()[:8])

	sessionReq := gateways.SessionRequest{
		Amount:             req.Amount,
		Currency:           req.Currency,
		SuccessRedirectURL: req.SuccessRedirectURL,
		CancelRedirectURL:  req.CancelRedirectURL,
	}

	// 2. Initialize session with gateway
	sessionResp, err := gateway.CreateSession(ctx, sessionReq)
	if err != nil {
		return nil, fmt.Errorf("gateway failed to create session: %w", err)
	}

	// 3. Persist transaction via repo
	tx := &PaymentTransaction{
		ID:              uuid.NewString(),
		ReferenceNumber: generatedRef,
		TaskID:          taskID,
		GatewayID:       req.GatewayID,
		SessionID:       sessionResp.SessionID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          PaymentStatusPending,
		ExpiryDate:      req.ExpiresAt,
		GatewayMetadata: req.Metadata,
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to persist transaction: %w", err)
	}

	return &CreateCheckoutResponse{
		ReferenceNumber: generatedRef,
		SessionID:       sessionResp.SessionID,
		Type:            sessionResp.Type,
		CheckoutURL:     sessionResp.CheckoutURL,
		Instructions:    sessionResp.Instructions,
		ExpiresIn:       int(time.Until(req.ExpiresAt).Seconds()),
	}, nil
}

func (s *paymentService) ValidateReference(ctx context.Context, gatewayID string, rawBody json.RawMessage) (*gateways.ValidationResponse, error) {
	slog.Info("validating incoming payment reference", "gateway", gatewayID)

	// 1. Get the gateway from the registry using the ID from the URL
	gateway, err := s.registry.Get(gatewayID)
	if err != nil {
		return nil, fmt.Errorf("gateway %s not found: %w", gatewayID, err)
	}

	// 2. Extract reference number from raw body
	refNo, err := gateway.ExtractReferenceNumber(ctx, rawBody)
	if err != nil {
		return nil, fmt.Errorf("failed to extract reference number: %w", err)
	}

	// 3. Look up the transaction metadata from the DB
	tx, err := s.repo.GetByReferenceNumber(ctx, refNo)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment reference: %w", err)
	}

	var validationTx gateways.ValidationTransaction
	if tx != nil {
		// 4. Security/Consistency check: ensure the transaction was actually intended for this gateway
		if tx.GatewayID != gatewayID {
			slog.Warn("gateway mismatch during validation", "expected", tx.GatewayID, "received", gatewayID, "reference", tx.ReferenceNumber)
			validationTx = gateways.ValidationTransaction{}
		} else {
			// 5. Map internal model to gateway DTO
			validationTx = gateways.ValidationTransaction{
				ReferenceNumber: tx.ReferenceNumber,
				Amount:          tx.Amount,
				Currency:        tx.Currency,
				Status:          string(tx.Status),
				ExpiryDate:      tx.ExpiryDate,
				Metadata:        tx.GatewayMetadata,
			}
		}
	}

	// 6. Delegate final validation response to the gateway
	return gateway.HandleValidateReference(ctx, validationTx, rawBody)
}

func (s *paymentService) ProcessWebhook(ctx context.Context, gatewayID string, body []byte, headers map[string][]string) error {
	gateway, err := s.registry.Get(gatewayID)
	if err != nil {
		return fmt.Errorf("failed to get gateway %s: %w", gatewayID, err)
	}

	gwPayload, err := gateway.ParseWebhook(ctx, body, headers)
	if err != nil {
		return fmt.Errorf("gateway failed to parse webhook: %w", err)
	}

	// 1. Look up transaction by ReferenceNumber
	tx, err := s.repo.GetByReferenceNumber(ctx, gwPayload.ReferenceNumber)
	if err != nil {
		return fmt.Errorf("failed to retrieve transaction by reference: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("transaction not found for reference: %s", gwPayload.ReferenceNumber)
	}

	// 2. Idempotency check: Ignore if we already recorded a final status
	if tx.Status == PaymentStatusSuccess || tx.Status == PaymentStatusFailed {
		slog.Info("webhook ignored (idempotent)", "reference", tx.ReferenceNumber, "current_status", tx.Status)
		return nil
	}

	// 3. Update status and metadata
	tx.Status = PaymentStatus(gwPayload.Status)
	tx.PaymentMethod = gwPayload.PaymentMethod
	if tx.GatewayMetadata == nil {
		tx.GatewayMetadata = make(map[string]string)
	}
	tx.GatewayMetadata["gateway_transaction_id"] = gwPayload.GatewayTransactionID
	tx.GatewayMetadata["webhook_timestamp"] = gwPayload.Timestamp

	if err := s.repo.Update(ctx, tx); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	slog.Info("processed webhook successfully", "reference", tx.ReferenceNumber, "status", tx.Status)

	// TODO: Emit internal events for the Task Engine
	// This would typically involve publishing to a message broker or a shared event bus.

	return nil
}
