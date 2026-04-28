package paymentsv2

import (
	"context"

	"github.com/OpenNSW/nsw/internal/paymentsv2/registry"
)

// PaymentRenderInfo is re-exported from registry package
type PaymentRenderInfo = registry.PaymentRenderInfo

// PaymentProviderInfo is the aggregate DTO used for provider discovery.
type PaymentProviderInfo struct {
	ID           string            `json:"id"`
	ProviderType string            `json:"provider_type"`
	IsActive     bool              `json:"is_active"`
	RenderInfo   PaymentRenderInfo `json:"render_info"`
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

	// GetByType retrieves a provider implementation by its configured provider type.
	GetByType(providerType string) (PaymentProvider, error)

	// ListInfo returns the aggregated metadata for all supported providers.
	ListInfo() []PaymentProviderInfo

	// GetDefault returns the primary provider implementation.
	GetDefault() (PaymentProvider, error)
}
