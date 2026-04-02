package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

type mockPaymentService struct {
	createCheckoutSessionFunc func(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)
	validateReferenceFunc     func(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error)
	processWebhookFunc        func(ctx context.Context, payload WebhookPayload) error
}

func (m *mockPaymentService) CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	return m.createCheckoutSessionFunc(ctx, req)
}

func (m *mockPaymentService) ValidateReference(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
	return m.validateReferenceFunc(ctx, req)
}

func (m *mockPaymentService) ProcessWebhook(ctx context.Context, payload WebhookPayload) error {
	return m.processWebhookFunc(ctx, payload)
}

func TestHandleValidateReference(t *testing.T) {
	service := &mockPaymentService{}
	handler := NewHTTPHandler(service)

	t.Run("success", func(t *testing.T) {
		service.validateReferenceFunc = func(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
			return &ValidateReferenceResponse{
				Amount:    decimal.NewFromFloat(100.0),
				IsPayable: true,
			}, nil
		}

		reqBody, _ := json.Marshal(ValidateReferenceRequest{PaymentReference: "REF-123"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/validate", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		handler.HandleValidateReference(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d", w.Code)
		}

		var resp ValidateReferenceResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if !resp.IsPayable {
			t.Error("expected IsPayable to be true")
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/validate", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.HandleValidateReference(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		service.validateReferenceFunc = func(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
			return nil, fmt.Errorf("service error")
		}

		reqBody, _ := json.Marshal(ValidateReferenceRequest{PaymentReference: "REF-123"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/validate", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		handler.HandleValidateReference(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status InternalServerError, got %d", w.Code)
		}
	})
}

func TestHandleWebhook(t *testing.T) {
	service := &mockPaymentService{}
	handler := NewHTTPHandler(service)

	t.Run("success", func(t *testing.T) {
		service.processWebhookFunc = func(ctx context.Context, payload WebhookPayload) error {
			return nil
		}

		payload := WebhookPayload{ReferenceNumber: "REF-123", Status: PaymentStatusSuccess}
		reqBody, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d", w.Code)
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		service.processWebhookFunc = func(ctx context.Context, payload WebhookPayload) error {
			return fmt.Errorf("service error")
		}

		payload := WebhookPayload{ReferenceNumber: "REF-123", Status: PaymentStatusSuccess}
		reqBody, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status InternalServerError, got %d", w.Code)
		}
	})
}
