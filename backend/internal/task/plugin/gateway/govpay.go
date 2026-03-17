package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

type GovPayProvider struct{}

func NewGovPayProvider() *GovPayProvider {
	return &GovPayProvider{}
}

func (p *GovPayProvider) ID() string {
	return "govpay"
}

func (p *GovPayProvider) GenerateRedirectURL(ctx context.Context, trx *payment_types.PaymentTransactionDB, returnUrl string) (string, error) {
	// In a real provider, we might use p.cfg to get the base URL
	baseURL := "https://checkout.govpay.lk"
	url := fmt.Sprintf("%s/pay?ref=%s&return=%s", baseURL, trx.ReferenceNumber, returnUrl)
	return url, nil
}

func (p *GovPayProvider) ExtractReference(r *http.Request) (string, error) {
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		return "", errors.New("missing ref query parameter")
	}
	return ref, nil
}

func (p *GovPayProvider) GetPaymentInfo(ctx context.Context, referenceNumber string) (CallbackResult, error) {
	return CallbackResult{
		ReferenceNumber: referenceNumber,
		ProviderID:      p.ID(),
		Status:          "SUCCESS",
	}, nil
}

func (p *GovPayProvider) FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error) {
	return map[string]interface{}{
		"confirmationNo": trx.ReferenceNumber,
		"paymentAmount":  trx.Amount,
		"currency":       "LKR",
		"status":         trx.Status,
		"provider":       p.ID(),
		"name":           trx.PayerName,
	}, nil
}
