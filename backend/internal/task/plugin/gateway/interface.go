package gateway

import (
	"context"
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
	// ID returns the unique identifier for the gateway, e.g. "govpay"
	ID() string

	// GenerateRedirectURL creates the secure checkout session URL to which the user is redirected.
	// It returns an error because generating a real session usually involves an outbound network call to the provider.
	GenerateRedirectURL(ctx context.Context, trx *payment_types.PaymentTransactionDB, returnUrl string) (string, error)

	// ExtractReference parses the incoming, unverified webhook request or redirect URL
	// to pluck out the reference number so the system knows which transaction to check.
	// This payload is untrusted.
	ExtractReference(r *http.Request) (string, error)

	// GetPaymentInfo makes a secure, outbound REST/API call to the gateway provider
	// using private backend API keys to determine the absolute source-of-truth status.
	GetPaymentInfo(ctx context.Context, referenceNumber string) (CallbackResult, error)

	// FormatInquiryResponse formats the database transaction record into the specific
	// JSON structure the gateway expects when it synchronously pings the Inquiry endpoint.
	FormatInquiryResponse(trx *payment_types.PaymentTransactionDB) (any, error)
}
