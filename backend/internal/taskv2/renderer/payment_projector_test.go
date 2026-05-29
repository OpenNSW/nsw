package renderer

import (
	"context"
	"errors"
	"testing"

	"github.com/OpenNSW/nsw/backend/internal/payments"
	"github.com/OpenNSW/nsw/backend/pkg/uiprojector"
	"github.com/stretchr/testify/assert"
)

type mockPaymentService struct {
	getPaymentMethodFunc func(id string) (*payments.PaymentMethod, error)
}

func (m *mockPaymentService) CreateCheckoutSession(ctx context.Context, req payments.CreateCheckoutRequest) (*payments.CreateCheckoutResponse, error) {
	return nil, nil
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
	return nil, errors.New("unimplemented")
}

func TestPaymentProjector_Project(t *testing.T) {
	mockSvc := &mockPaymentService{}
	proj := NewPaymentProjector(mockSvc)

	t.Run("renders lankapay redirect template", func(t *testing.T) {
		mockSvc.getPaymentMethodFunc = func(id string) (*payments.PaymentMethod, error) {
			assert.Equal(t, "lankapay", id)
			return &payments.PaymentMethod{
				ID:       id,
				Type:     "REDIRECT",
				Template: "LankaPay {{ .ServiceName }} ({{ .CheckoutURL }})",
			}, nil
		}

		data := map[string]any{
			"selected_method": "lankapay",
			"checkout_url":    "https://checkout.lk/123",
			"service_name":    "App Fee",
		}

		out, err := proj.Project(context.Background(), nil, data)
		assert.NoError(t, err)
		assert.Equal(t, uiprojector.SectionType("REDIRECT"), out.Type)
		content, ok := out.Content.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "https://checkout.lk/123", content["checkout_url"])
		assert.Equal(t, "LankaPay App Fee (https://checkout.lk/123)", content["content"])
	})

	t.Run("renders govpay offline template", func(t *testing.T) {
		mockSvc.getPaymentMethodFunc = func(id string) (*payments.PaymentMethod, error) {
			assert.Equal(t, "govpay", id)
			return &payments.PaymentMethod{
				ID:       id,
				Type:     "INFO",
				Template: "GovPay {{ .ReferenceNumber }} amount {{ .Amount }}",
			}, nil
		}

		data := map[string]any{
			"selected_method":  "govpay",
			"reference_number": "REF-999",
			"amount":           "1500.00",
		}

		out, err := proj.Project(context.Background(), nil, data)
		assert.NoError(t, err)
		assert.Equal(t, uiprojector.SectionTypeMarkdown, out.Type)
		assert.Equal(t, "GovPay REF-999 amount 1500.00", out.Content)
	})
}
