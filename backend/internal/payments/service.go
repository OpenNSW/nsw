package payments

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
)

// TaskCompleter defines the interface to resume suspended workflow steps.
type TaskCompleter interface {
	CompleteTaskStep(ctx context.Context, taskID string, payload map[string]any) error
}

// PaymentMethod holds the metadata and configuration for a payment gateway/method.
type PaymentMethod struct {
	ID         string `json:"id"`
	IsActive   bool   `json:"is_active"`
	Type       string `json:"type"`
	GatewayURL string `json:"gateway_url"`
	Template   string `json:"template"`
}

type paymentMethodsConfig struct {
	Version string          `json:"version"`
	Methods []PaymentMethod `json:"methods"`
}

// PaymentService defines the business logic operations for Payments.
type PaymentService interface {
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)
	ValidateReference(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error)
	ProcessWebhook(ctx context.Context, payload WebhookPayload) error
	SetTaskCompleter(completer TaskCompleter)
	GetPaymentMethod(id string) (*PaymentMethod, error)
}

type paymentService struct {
	repo          PaymentRepository
	taskCompleter TaskCompleter
	methods       []PaymentMethod
	methodIdx     map[string]int
}

// NewPaymentService loads payment methods from configPath and initializes a new payment service.
func NewPaymentService(repo PaymentRepository, configPath string) (PaymentService, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("payments: read config %q: %w", configPath, err)
	}
	var cfg paymentMethodsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("payments: parse config %q: %w", configPath, err)
	}
	idx := make(map[string]int, len(cfg.Methods))
	for i, m := range cfg.Methods {
		idx[m.ID] = i
	}
	return &paymentService{repo: repo, methods: cfg.Methods, methodIdx: idx}, nil
}

func (s *paymentService) SetTaskCompleter(completer TaskCompleter) {
	s.taskCompleter = completer
}

func (s *paymentService) GetPaymentMethod(id string) (*PaymentMethod, error) {
	i, ok := s.methodIdx[id]
	if !ok {
		return nil, fmt.Errorf("payment method %q not found", id)
	}
	return &s.methods[i], nil
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generatePaymentReference() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return "TNSW" + string(b)
}

// CreateCheckoutSession saves the initial intent and returns mocked LankaPay session details.
func (s *paymentService) CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	sessionID := "sess_" + uuid.NewString()
	taskID, ok := req.Metadata["task_id"]
	if !ok {
		return nil, fmt.Errorf("task_id is required in metadata")
	}

	refNum := req.ReferenceNumber
	if refNum == "" {
		for {
			candidate := generatePaymentReference()
			existing, err := s.repo.GetByReferenceNumber(ctx, candidate)
			if err != nil {
				return nil, fmt.Errorf("failed to check existing reference number: %w", err)
			}
			if existing == nil {
				refNum = candidate
				break
			}
		}
	}

	tx := &PaymentTransaction{
		ID:              uuid.NewString(),
		ReferenceNumber: refNum,
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

	slog.Info("created checkout session", "reference_number", refNum, "session_id", sessionID)

	return &CreateCheckoutResponse{
		SessionID:       sessionID,
		CheckoutURL:     "https://sandbox.govpay.lk/checkout/" + sessionID,
		ExpiresIn:       int(time.Until(req.ExpiresAt).Seconds()),
		ReferenceNumber: refNum,
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

	if s.taskCompleter != nil {
		statusStr := "success"
		if tx.Status == PaymentStatusFailed {
			statusStr = "fail"
		}
		slog.Info("taskv2 payment: webhook advancing task step", "taskId", tx.TaskID, "status", statusStr)
		if err := s.taskCompleter.CompleteTaskStep(ctx, tx.TaskID, map[string]any{"payment_status": statusStr}); err != nil {
			slog.Error("taskv2 payment: failed to automatically advance task step", "taskId", tx.TaskID, "error", err)
		}
	}

	return nil
}
