package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubProvider struct{ name string }

func (p *stubProvider) CreateSession(_ context.Context, _ interface{}) (interface{}, error) {
	return &CreateCheckoutResponse{ReferenceNumber: "ref", SessionID: p.name, CheckoutURL: "https://example.com"}, nil
}

func (p *stubProvider) ParseWebhook(_ context.Context, _ []byte, _ map[string][]string) (interface{}, error) {
	return &WebhookPayload{ReferenceNumber: p.name}, nil
}

func (p *stubProvider) HandleValidateReference(_ context.Context, _ interface{}) (interface{}, error) {
	return &ValidateReferenceResponse{IsPayable: true}, nil
}

type CreateCheckoutResponse struct {
	ReferenceNumber string `json:"reference_number"`
	SessionID       string `json:"session_id"`
	CheckoutURL     string `json:"checkout_url"`
	ExpiresIn       int    `json:"expires_in_seconds"`
}

type WebhookPayload struct {
	ReferenceNumber string `json:"reference_number"`
}

type ValidateReferenceResponse struct {
	IsPayable bool `json:"is_payable"`
}

func writeConfigFile(t *testing.T, dir string, json string) string {
	t.Helper()
	path := filepath.Join(dir, "payment-options.json")
	require.NoError(t, os.WriteFile(path, []byte(json), 0o600))
	return path
}

func TestConfiguredLoadsFromJSONAndResolvesByIDAndType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeConfigFile(t, dir, `{
		"version": "1.0",
		"default_id": "card",
		"options": [
			{
				"id": "wallet",
				"type": "redirect",
				"is_active": true,
				"render_info": {
					"display_name": "Wallet",
					"description": "Wallet payments",
					"logo_url": "wallet.svg",
					"display_order": 2
				}
			},
			{
				"id": "card",
				"provider_type": "redirect",
				"is_active": true,
				"render_info": {
					"display_name": "Card",
					"description": "Card payments",
					"logo_url": "card.svg",
					"display_order": 1
				}
			}
		]
	}`)

	registry := NewConfigured()
	registry.RegisterProvider("redirect", &stubProvider{name: "redirect-provider"})
	require.NoError(t, registry.LoadFromFile(path))

	defaultProvider, err := registry.GetDefault()
	require.NoError(t, err)
	require.NotNil(t, defaultProvider)

	providerByID, err := registry.Get("card")
	require.NoError(t, err)
	require.Same(t, defaultProvider, providerByID)

	providerByType, err := registry.GetByType("redirect")
	require.NoError(t, err)
	require.Same(t, defaultProvider, providerByType)

	infos := registry.ListInfo()
	require.Len(t, infos, 2)
	require.Equal(t, "card", infos[0].ID)
	require.Equal(t, "redirect", infos[0].ProviderType)
	require.Equal(t, "wallet", infos[1].ID)
	require.Equal(t, "redirect", infos[1].ProviderType)
}

func TestConfiguredRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	registry := NewConfigured()
	err := registry.LoadConfig(Config{
		Version: "1.0",
		Options: []PaymentOptionConfig{
			{ID: "dup", ProviderType: "alpha", IsActive: true, RenderInfo: PaymentRenderInfo{DisplayName: "A", Description: "A", DisplayOrder: 1}},
			{ID: "dup", ProviderType: "beta", IsActive: true, RenderInfo: PaymentRenderInfo{DisplayName: "B", Description: "B", DisplayOrder: 2}},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestConfiguredRejectsInvalidDefault(t *testing.T) {
	t.Parallel()

	registry := NewConfigured()
	registry.RegisterProvider("alpha", &stubProvider{name: "alpha"})

	err := registry.LoadConfig(Config{
		Version:   "1.0",
		DefaultID: "inactive",
		Options: []PaymentOptionConfig{
			{ID: "active", ProviderType: "alpha", IsActive: true, RenderInfo: PaymentRenderInfo{DisplayName: "A", Description: "A", DisplayOrder: 1}},
			{ID: "inactive", ProviderType: "alpha", IsActive: false, RenderInfo: PaymentRenderInfo{DisplayName: "I", Description: "I", DisplayOrder: 2}},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be enabled")
}

func TestConfiguredRejectsUnsupportedEnabledProviderType(t *testing.T) {
	t.Parallel()

	registry := NewConfigured()
	err := registry.LoadConfig(Config{
		Version: "1.0",
		Options: []PaymentOptionConfig{
			{ID: "active", ProviderType: "missing", IsActive: true, RenderInfo: PaymentRenderInfo{DisplayName: "A", Description: "A", DisplayOrder: 1}},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported provider type")
}
