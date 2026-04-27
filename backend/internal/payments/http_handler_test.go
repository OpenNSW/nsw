package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPaymentService struct {
	PaymentService
	validateRes *ValidateReferenceResponse
	validateErr error
	webhookErr  error
}

func (m *mockPaymentService) ValidateReference(ctx context.Context, providerID string, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
	return m.validateRes, m.validateErr
}

func (m *mockPaymentService) ProcessWebhook(ctx context.Context, providerID string, body []byte, headers map[string][]string) error {
	return m.webhookErr
}

func TestHTTPHandler_HandleValidateReference(t *testing.T) {
	service := &mockPaymentService{}
	handler := NewHTTPHandler(service)

	t.Run("success", func(t *testing.T) {
		service.validateRes = &ValidateReferenceResponse{IsPayable: true}
		reqBody, _ := json.Marshal(ValidateReferenceRequest{PaymentReference: "REF-123"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/validate?provider=lankapay", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		handler.HandleValidateReference(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d", w.Code)
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
}

func TestHTTPHandler_HandleWebhook(t *testing.T) {
	service := &mockPaymentService{}
	handler := NewHTTPHandler(service)

	t.Run("success", func(t *testing.T) {
		service.webhookErr = nil
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook?provider=lankapay", bytes.NewBufferString("payload"))
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d", w.Code)
		}
	})
}
