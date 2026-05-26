package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/OpenNSW/nsw-task-flow/store"
	"github.com/OpenNSW/nsw/internal/payments"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type mockPaymentService struct {
	createCheckoutSessionFunc func(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error)
	getPaymentMethodFunc      func(id string) (*payments.PaymentMethod, error)
}

func (m *mockPaymentService) CreateCheckoutSession(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error) {
	if m.createCheckoutSessionFunc != nil {
		return m.createCheckoutSessionFunc(ctx, req)
	}
	return nil, errors.New("unimplemented")
}

func (m *mockPaymentService) ValidateReference(ctx context.Context, req payments.ValidateReferenceRequest) (*payments.ValidateReferenceResponse, error) {
	return nil, nil
}

func (m *mockPaymentService) ProcessWebhook(ctx context.Context, payload payments.WebhookPayload) error {
	return nil
}

func (m *mockPaymentService) SetTaskCompleter(completer payments.TaskCompleter) {}

func (m *mockPaymentService) GetPaymentMethod(id string) (*payments.PaymentMethod, error) {
	if m.getPaymentMethodFunc != nil {
		return m.getPaymentMethodFunc(id)
	}
	return &payments.PaymentMethod{
		ID:         id,
		IsActive:   true,
		Type:       "REDIRECT",
		GatewayURL: "https://sandbox.govpay.lk/checkout",
		Template:   "# Mock Template",
	}, nil
}

func TestPaymentPlugin_Execute(t *testing.T) {
	// Setup
	mockSvc := &mockPaymentService{}
	plugin := NewPaymentPlugin(mockSvc)

	// Check config
	configRaw := json.RawMessage(`{"task_code": "fcau_app_fee_payment_v1"}`)

	t.Run("successful execution", func(t *testing.T) {
		record := store.TaskRecord{
			TaskID:                "test-task-123",
			ActiveOutputNamespace: "payment",
		}

		mockSvc.createCheckoutSessionFunc = func(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error) {
			assert.Equal(t, "", req.ReferenceNumber)
			assert.True(t, req.Amount.Equal(decimal.NewFromFloat(1500.00)))
			assert.Equal(t, "LKR", req.Currency)
			assert.Equal(t, "test-task-123", req.Metadata["task_id"])
			assert.Equal(t, "fcau_app_fee_payment_v1", req.Metadata["task_code"])

			return &payments.CreateCheckoutResponse{
				SessionID:       "mock-session-123",
				CheckoutURL:     "https://sandbox.govpay.lk/checkout/mock-session-123",
				ReferenceNumber: "TNSW-MOCK123",
			}, nil
		}

		ctx := pluginContext{
			Context: context.Background(),
			Record:  &record,
			Inputs:  map[string]any{}, // empty inputs to trigger fallback
		}

		err := plugin.Execute(ctx, configRaw)
		assert.True(t, errors.Is(err, ErrSuspended), "expected ErrSuspended")
		assert.Equal(t, "PENDING_PAYMENT", record.State)

		// Check output data structure
		paymentData, ok := record.Data["payment"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "mock-session-123", paymentData["session_id"])
		assert.Equal(t, "https://sandbox.govpay.lk/checkout/mock-session-123", paymentData["checkout_url"])
		assert.Equal(t, "TNSW-MOCK123", paymentData["reference_number"])
		assert.Equal(t, "1500", paymentData["amount"])
		assert.Equal(t, "LKR", paymentData["currency"])
	})

	t.Run("successful execution with explicit reference number", func(t *testing.T) {
		record := store.TaskRecord{
			TaskID:                "test-task-123",
			ActiveOutputNamespace: "payment",
		}

		mockSvc.createCheckoutSessionFunc = func(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error) {
			assert.Equal(t, "", req.ReferenceNumber)
			return &payments.CreateCheckoutResponse{
				SessionID:       "mock-session-999",
				ReferenceNumber: "TNSW-MOCK999",
			}, nil
		}

		ctx := pluginContext{
			Context: context.Background(),
			Record:  &record,
			Inputs: map[string]any{
				"reference_number": "NSW-REF-999",
			},
		}

		err := plugin.Execute(ctx, configRaw)
		assert.True(t, errors.Is(err, ErrSuspended))
	})

	t.Run("successful execution with govpay offline method selection", func(t *testing.T) {
		record := store.TaskRecord{
			TaskID:                "test-task-123",
			ActiveOutputNamespace: "payment",
		}

		mockSvc.getPaymentMethodFunc = func(id string) (*payments.PaymentMethod, error) {
			assert.Equal(t, "govpay", id)
			return &payments.PaymentMethod{
				ID:       id,
				IsActive: true,
				Type:     "INFO",
				Template: "Pay LKR {{ .Amount }} for {{ .ReferenceNumber }}",
			}, nil
		}

		mockSvc.createCheckoutSessionFunc = func(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error) {
			assert.Equal(t, "", req.ReferenceNumber)
			return &payments.CreateCheckoutResponse{
				SessionID:       "mock-govpay-session",
				ReferenceNumber: "TNSW-MOCKGOV",
			}, nil
		}

		ctx := pluginContext{
			Context: context.Background(),
			Record:  &record,
			Inputs: map[string]any{
				"reference_number": "NSW-REF-GOVPAY",
				"selected_method":  "govpay",
			},
		}

		err := plugin.Execute(ctx, configRaw)
		assert.True(t, errors.Is(err, ErrSuspended))

		// Check payment details were saved correctly
		paymentData, ok := record.Data["payment"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "govpay", paymentData["selected_method"])
		assert.Equal(t, "Application Fee", paymentData["service_name"])
		assert.Equal(t, "fcau_app_fee_payment_v1", paymentData["service_type"])
		assert.Equal(t, "TNSW-MOCKGOV", paymentData["reference_number"])
	})
}
