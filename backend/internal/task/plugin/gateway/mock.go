package gateway

import (
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

func (p *MockGateway) GenerateRedirectURL(referenceNumber string) string {
	return fmt.Sprintf("http://localhost:5173/mock-payment?ref=%s", referenceNumber)
}

func (p *MockGateway) VerifyCallback(r *http.Request) (CallbackResult, error) {
	var payload struct {
		ReferenceNumber string `json:"reference_number"`
		Status          string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return CallbackResult{}, fmt.Errorf("failed to decode mock payload: %w", err)
	}

	return CallbackResult{
		ReferenceNumber: payload.ReferenceNumber,
		ProviderID:      p.ID(),
		Status:          payload.Status,
	}, nil
}

func (p *MockGateway) FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error) {
	return map[string]interface{}{
		"provider":         p.ID(),
		"reference_number": trx.ReferenceNumber,
		"amount":           trx.Amount,
		"currency":         "MOCK",
		"status":           trx.Status,
	}, nil
}
