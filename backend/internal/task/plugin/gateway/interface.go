package gateway

import (
	"net/http"

	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
)

// CallbackResult standardizes what the generic PaymentHandler needs to know
// to transition the FSM, regardless of which gateway sent the webhook.
type CallbackResult struct {
	ReferenceNumber string
	ProviderID      string
	Status          string // "SUCCESS" or "FAILED"
}

// PaymentGateway defines the interface all external payment providers must implement.
type PaymentGateway interface {
	// ID returns the unique identifier for the gateway (e.g., "govpay", "mock").
	ID() string

	// GenerateRedirectURL creates the URL the user clicks to pay.
	GenerateRedirectURL(referenceNumber string) string

	// VerifyCallback validates the webhook signature and extracts standard result data.
	VerifyCallback(r *http.Request) (CallbackResult, error)

	// FormatInquiryResponse formats the DB transaction into the specific JSON
	// structure the gateway expects when it pings the Inquiry endpoint.
	FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error)
}
