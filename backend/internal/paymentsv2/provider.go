package paymentsv2

import (
	"context"
)

// PaymentRenderInfo contains UI-specific metadata for displaying a payment method.
type PaymentRenderInfo struct {
	DisplayName  string `json:"display_name"`
	Description  string `json:"description"`
	LogoURL      string `json:"logo_url"`
	DisplayOrder int    `json:"display_order"`
	PrimaryColor string `json:"primary_color,omitempty"`
}

// PaymentProviderInfo is the aggregate DTO used for provider discovery.
type PaymentProviderInfo struct {
	ID         string            `json:"id"`
	IsActive   bool              `json:"is_active"`
	RenderInfo PaymentRenderInfo `json:"render_info"`
}

// PaymentProvider defines the interface for external payment gateway integration.
type PaymentProvider interface {
	// CreateSession initializes a payment session with the gateway.
	CreateSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)

	// ParseWebhook processes raw gateway notifications into domain-neutral payloads.
	ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (*WebhookPayload, error)

	// HandleValidateReference handles gateway-specific validation logic.
	// This is called when a gateway queries if a reference is valid and payable.
	HandleValidateReference(ctx context.Context, tx *PaymentTransaction) (*ValidateReferenceResponse, error)
}

// PaymentRegistry manages the discovery and lookup of payment providers.
type PaymentRegistry interface {
	// Get retrieves a provider implementation by its ID.
	Get(id string) (PaymentProvider, error)

	// ListInfo returns the aggregated metadata for all supported providers.
	ListInfo() []PaymentProviderInfo

	// GetDefault returns the primary provider implementation.
	GetDefault() (PaymentProvider, error)
}
