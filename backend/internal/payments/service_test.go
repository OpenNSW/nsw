package payments

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type mockRepository struct {
	txs       map[string]*PaymentTransaction
	createErr error
	getErr    error
	updateErr error
}

func (m *mockRepository) Create(ctx context.Context, tx *PaymentTransaction) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.txs[tx.ReferenceNumber] = tx
	return nil
}

func (m *mockRepository) GetByReferenceNumber(ctx context.Context, ref string) (*PaymentTransaction, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if tx, ok := m.txs[ref]; ok {
		return tx, nil
	}
	return nil, nil
}

func (m *mockRepository) GetByTaskID(ctx context.Context, taskID string) (*PaymentTransaction, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, tx := range m.txs {
		if tx.TaskID == taskID {
			return tx, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, tx *PaymentTransaction) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.txs[tx.ReferenceNumber] = tx
	return nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, ref string, status PaymentStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if tx, ok := m.txs[ref]; ok {
		tx.Status = status
	}
	return nil
}

func (m *mockRepository) WithTx(tx *gorm.DB) PaymentRepository {
	return m
}

func TestCreateCheckoutSession(t *testing.T) {
	repo := &mockRepository{txs: make(map[string]*PaymentTransaction)}
	service := NewPaymentService(repo)

	req := CreateCheckoutRequest{
		ReferenceNumber: "REF-123",
		Amount:          decimal.NewFromFloat(100.0),
		Currency:        "LKR",
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		Metadata:        map[string]string{"task_id": "TASK-123"},
	}

	t.Run("success", func(t *testing.T) {
		resp, err := service.CreateCheckoutSession(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.SessionID == "" {
			t.Fatal("expected session ID to be generated")
		}
	})

	t.Run("missing task_id", func(t *testing.T) {
		reqMissing := req
		reqMissing.Metadata = nil
		_, err := service.CreateCheckoutSession(context.Background(), reqMissing)
		if err == nil {
			t.Fatal("expected error for missing task_id, got nil")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo.createErr = fmt.Errorf("db error")
		_, err := service.CreateCheckoutSession(context.Background(), req)
		if err == nil {
			t.Fatal("expected error for repo failure, got nil")
		}
		repo.createErr = nil
	})
}

func TestValidateReference(t *testing.T) {
	repo := &mockRepository{txs: make(map[string]*PaymentTransaction)}
	service := NewPaymentService(repo)

	t.Run("not found", func(t *testing.T) {
		resp, err := service.ValidateReference(context.Background(), ValidateReferenceRequest{PaymentReference: "NON-EXISTENT"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.IsPayable {
			t.Fatal("expected IsPayable to be false for non-existent reference")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo.getErr = fmt.Errorf("db error")
		_, err := service.ValidateReference(context.Background(), ValidateReferenceRequest{PaymentReference: "REF-123"})
		if err == nil {
			t.Fatal("expected error for repo failure, got nil")
		}
		repo.getErr = nil
	})

	t.Run("success pending", func(t *testing.T) {
		expiry := time.Now().Add(1 * time.Hour)
		repo.txs["REF-123"] = &PaymentTransaction{
			ReferenceNumber: "REF-123",
			Status:          PaymentStatusPending,
			Amount:          decimal.NewFromFloat(100.0),
			Currency:        "LKR",
			ExpiryDate:      expiry,
		}

		resp, err := service.ValidateReference(context.Background(), ValidateReferenceRequest{PaymentReference: "REF-123"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.IsPayable {
			t.Fatal("expected IsPayable to be true for pending reference")
		}
	})
}

func TestProcessWebhook(t *testing.T) {
	repo := &mockRepository{txs: make(map[string]*PaymentTransaction)}
	service := NewPaymentService(repo)

	txKey := "REF-123"
	repo.txs[txKey] = &PaymentTransaction{
		ReferenceNumber: txKey,
		Status:          PaymentStatusPending,
		Amount:          decimal.NewFromFloat(100.0),
		Currency:        "LKR",
		ExpiryDate:      time.Now().Add(1 * time.Hour),
	}

	t.Run("success", func(t *testing.T) {
		payload := WebhookPayload{
			ReferenceNumber:      txKey,
			GatewayTransactionID: "GW-123",
			Status:               PaymentStatusSuccess,
			PaymentMethod:        "CC",
			Timestamp:            time.Now().Format(time.RFC3339),
		}

		err := service.ProcessWebhook(context.Background(), payload)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.txs[txKey].Status != PaymentStatusSuccess {
			t.Errorf("expected status SUCCESS, got %s", repo.txs[txKey].Status)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := service.ProcessWebhook(context.Background(), WebhookPayload{ReferenceNumber: "UNKNOWN"})
		if err == nil {
			t.Fatal("expected error for unknown reference, got nil")
		}
	})

	t.Run("repo error get", func(t *testing.T) {
		repo.getErr = fmt.Errorf("db error")
		err := service.ProcessWebhook(context.Background(), WebhookPayload{ReferenceNumber: txKey})
		if err == nil {
			t.Fatal("expected error for repo failure, got nil")
		}
		repo.getErr = nil
	})

	t.Run("repo error update", func(t *testing.T) {
		txKeyErr := "REF-ERR-UPD"
		repo.txs[txKeyErr] = &PaymentTransaction{
			ReferenceNumber: txKeyErr,
			Status:          PaymentStatusPending,
		}
		repo.updateErr = fmt.Errorf("db error")
		payload := WebhookPayload{
			ReferenceNumber: txKeyErr,
			Status:          PaymentStatusSuccess,
		}
		err := service.ProcessWebhook(context.Background(), payload)
		if err == nil {
			t.Fatal("expected error for repo failure, got nil")
		}
		repo.updateErr = nil
	})

	t.Run("idempotency", func(t *testing.T) {
		txKeyIdem := "REF-IDEM"
		repo.txs[txKeyIdem] = &PaymentTransaction{
			ReferenceNumber: txKeyIdem,
			Status:          PaymentStatusSuccess,
		}
		payload := WebhookPayload{
			ReferenceNumber: txKeyIdem,
			Status:          PaymentStatusSuccess,
		}
		err := service.ProcessWebhook(context.Background(), payload)
		if err != nil {
			t.Fatalf("expected no error for idempotent call, got %v", err)
		}
	})
}
