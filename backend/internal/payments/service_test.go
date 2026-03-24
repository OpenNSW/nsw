package payments

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type mockRepository struct {
	txs map[string]*PaymentTransaction
}

func (m *mockRepository) Create(ctx context.Context, tx *PaymentTransaction) error {
	m.txs[tx.ReferenceNumber] = tx
	return nil
}

func (m *mockRepository) GetByReferenceNumber(ctx context.Context, ref string) (*PaymentTransaction, error) {
	if tx, ok := m.txs[ref]; ok {
		return tx, nil
	}
	return nil, nil // Return nil with no error if not found, matching repository interface
}

func (m *mockRepository) GetByTaskID(ctx context.Context, taskID string) (*PaymentTransaction, error) {
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, tx *PaymentTransaction) error {
	m.txs[tx.ReferenceNumber] = tx
	return nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, ref string, status string) error {
	if tx, ok := m.txs[ref]; ok {
		tx.Status = status
	}
	return nil
}

func (m *mockRepository) WithTx(tx *gorm.DB) PaymentRepository {
	return m // Mock implementation, ignores transactions
}

func TestProcessWebhook_Idempotency(t *testing.T) {
	repo := &mockRepository{txs: make(map[string]*PaymentTransaction)}
	service := NewPaymentService(repo) // Use internal struct to avoid dependency cycles if in a separate package, but we are in same package

	// Seed an existing pending transaction
	txKey := "REF-123"
	repo.txs[txKey] = &PaymentTransaction{
		ID:              uuid.New(),
		ReferenceNumber: txKey,
		Status:          "PENDING",
		Amount:          decimal.NewFromFloat(100.0),
		Currency:        "LKR",
		ExpiryDate:      time.Now().Add(1 * time.Hour),
	}

	payload := WebhookPayload{
		ReferenceNumber:      txKey,
		GatewayTransactionID: "GW-456",
		Status:               "SUCCESS",
		PaymentMethod:        "BANK_TRANSFER",
		Timestamp:            "2026-03-24T14:09:00Z",
	}

	// 1. Process first webhook
	err := service.ProcessWebhook(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.txs[txKey].Status != "SUCCESS" {
		t.Fatalf("expected status to be SUCCESS, got %s", repo.txs[txKey].Status)
	}
	if repo.txs[txKey].GatewayMetadata["gateway_transaction_id"] != "GW-456" {
		t.Fatalf("expected gateway metadata to be populated")
	}

	// 2. Process duplicate webhook
	err = service.ProcessWebhook(context.Background(), payload)
	if err != nil {
		t.Fatalf("expected no error on duplicate webhook, got %v", err)
	}
}
