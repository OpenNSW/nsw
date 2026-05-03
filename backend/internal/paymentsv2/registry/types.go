package registry

import "context"

// PaymentProvider represents the provider interface
type PaymentProvider interface {
	CreateSession(ctx context.Context, req interface{}) (interface{}, error)
	ParseWebhook(ctx context.Context, body []byte, headers map[string][]string) (interface{}, error)
	HandleValidateReference(ctx context.Context, tx interface{}) (interface{}, error)
}

// PaymentProviderInfo is the metadata for a payment provider
type PaymentProviderInfo struct {
	ID           string            `json:"id"`
	ProviderType string            `json:"provider_type"`
	IsActive     bool              `json:"is_active"`
	RenderInfo   PaymentRenderInfo `json:"render_info"`
}
