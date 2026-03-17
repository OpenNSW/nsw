package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

type MockGateway struct{}

func NewMockGateway() *MockGateway {
	return &MockGateway{}
}

func (p *MockGateway) ID() string {
	return "mock"
}

func (p *MockGateway) GenerateRedirectURL(ctx context.Context, trx *payment_types.PaymentTransactionDB, returnUrl string) (string, error) {
	// In mock mode, we often return an empty string to trigger the integrated frontend dialog
	return "", nil
}

func (p *MockGateway) ExtractReference(r *http.Request) (string, error) {
	var payload struct {
		ReferenceNumber string `json:"reference_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode mock payload: %w", err)
	}

	return payload.ReferenceNumber, nil
}

func (p *MockGateway) GetPaymentInfo(ctx context.Context, referenceNumber string) (CallbackResult, error) {
	return CallbackResult{
		ReferenceNumber: referenceNumber,
		ProviderID:      p.ID(),
		Status:          "SUCCESS",
	}, nil
}

func (p *MockGateway) FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error) {
	return map[string]interface{}{
		"confirmationNo": trx.ReferenceNumber,
		"paymentAmount":  trx.Amount,
		"currency":       "MOCK",
		"status":         trx.Status,
		"provider":       p.ID(),
		"name":           trx.PayerName,
	}, nil
}
