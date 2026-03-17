package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

type GovPayProvider struct {
	cfg *config.Config
}

func NewGovPayProvider(cfg *config.Config) *GovPayProvider {
	return &GovPayProvider{cfg: cfg}
}

func (p *GovPayProvider) ID() string {
	return "govpay"
}

func (p *GovPayProvider) GenerateRedirectURL(referenceNumber string) string {
	// Note: In a real system, we might have a specific redirect URL per gateway.
	// For now, we use a hardcoded production default or mock mode.
	return fmt.Sprintf("https://www.lpopp.lk/pay?ref=%s", referenceNumber)
}

func (p *GovPayProvider) VerifyCallback(r *http.Request) (CallbackResult, error) {
	// GovPay specific signature check (Placeholder)
	if !p.cfg.Payment.MockMode {
		signature := r.Header.Get("GovPay-Signature")
		if signature == "" {
			return CallbackResult{}, errors.New("missing govpay signature")
		}
		// TODO: verifySignature(r.Body, signature, p.cfg.Payment.SecretKey)
	}

	// GovPay specific JSON decoding
	var payload struct {
		ReferenceNumber string `json:"reference_number"`
		Status          string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return CallbackResult{}, fmt.Errorf("failed to decode govpay payload: %w", err)
	}

	return CallbackResult{
		ReferenceNumber: payload.ReferenceNumber,
		ProviderID:      p.ID(),
		Status:          payload.Status,
	}, nil
}

func (p *GovPayProvider) FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error) {
	return map[string]interface{}{
		"provider":         p.ID(),
		"reference_number": trx.ReferenceNumber,
		"amount":           trx.Amount,
		"currency":         "LKR",
		"status":           trx.Status,
	}, nil
}
