package gateways

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type InteractionType string

const (
	FlowTypeRedirect    InteractionType = "REDIRECT"
	FlowTypeInstruction InteractionType = "INSTRUCTION"
)

type SessionRequest struct {
	Amount             string `json:"amount"`
	Currency           string `json:"currency"`
	SuccessRedirectURL string `json:"success_redirect_url"`
	CancelRedirectURL  string `json:"cancel_redirect_url"`
}

type SessionResponse struct {
	SessionID    string          `json:"session_id"`
	Type         InteractionType `json:"type"`
	CheckoutURL  string          `json:"checkout_url,omitempty"`
	Instructions string          `json:"instructions,omitempty"`
}

// WebhookPayload represents the external callback from LankaPay to the Payment Service.
type WebhookPayload struct {
	ReferenceNumber      string            `json:"reference_number"`
	SessionID            string            `json:"session_id"`
	GatewayTransactionID string            `json:"gateway_transaction_id"`
	Status               string            `json:"status"`
	Amount               decimal.Decimal   `json:"amount"`
	Currency             string            `json:"currency"`
	PaymentMethod        string            `json:"payment_method"`
	Timestamp            string            `json:"timestamp"`
	Metadata             map[string]string `json:"metadata"`
}

// ValidationTransaction represents a minimal view of a payment transaction for validation purposes.
type ValidationTransaction struct {
	ReferenceNumber string            `json:"reference_number"`
	Amount          decimal.Decimal   `json:"amount"`
	Currency        string            `json:"currency"`
	Status          string            `json:"status"`
	ExpiryDate      time.Time         `json:"expiry_date"`
	Metadata        map[string]string `json:"metadata"`
}

// ValidationResponse represents a structured response for a validation request.
type ValidationResponse struct {
	Payload    json.RawMessage
	HTTPStatus int
}

// PaymentGateway defines the interface for external payment gateway integration.
type PaymentGateway interface {
	// ApplyConfig injects the configuration for the gateway method.
	ApplyConfig(config json.RawMessage) error

	// GetFlowType returns the flow type of the gateway (REDIRECT or INSTRUCTION).
	GetFlowType() InteractionType

	// CreateSession initializes a payment session with the gateway.
	CreateSession(ctx context.Context, req SessionRequest) (*SessionResponse, error)

	// ExtractReferenceNumber parses the gateway-specific validation request to extract the reference number.
	ExtractReferenceNumber(ctx context.Context, reqData json.RawMessage) (string, error)

	// HandleValidateReference handles the final validation response after the transaction is found.
	HandleValidateReference(ctx context.Context, tx ValidationTransaction, reqData json.RawMessage) (*ValidationResponse, error)

	// ParseWebhook processes raw gateway notifications into domain-neutral payloads.
	ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (*WebhookPayload, error)
}
