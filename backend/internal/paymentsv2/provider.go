package paymentsv2

import (
	"github.com/OpenNSW/nsw/internal/paymentsv2/registry"
)

// PaymentRenderInfo is re-exported from registry package
type PaymentRenderInfo = registry.PaymentRenderInfo

// PaymentProviderInfo is re-exported from the registry package
type PaymentProviderInfo = registry.PaymentProviderInfo

// PaymentProvider is re-exported from the registry package. Implementations
// should satisfy the registry.PaymentProvider contract.
type PaymentProvider = registry.PaymentProvider

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
