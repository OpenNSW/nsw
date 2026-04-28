package registry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// PaymentRenderInfo contains UI-specific metadata for displaying a payment method.
type PaymentRenderInfo struct {
	DisplayName  string `json:"display_name"`
	Description  string `json:"description"`
	LogoURL      string `json:"logo_url"`
	DisplayOrder int    `json:"display_order"`
	PrimaryColor string `json:"primary_color,omitempty"`
}

// PaymentOptionConfig defines a single payment option entry in the registry JSON schema.
type PaymentOptionConfig struct {
	ID           string            `json:"id"`
	ProviderType string            `json:"provider_type"`
	IsActive     bool              `json:"is_active"`
	RenderInfo   PaymentRenderInfo `json:"render_info"`
}

// UnmarshalJSON accepts both `provider_type` and legacy `type` fields.
func (c *PaymentOptionConfig) UnmarshalJSON(data []byte) error {
	type rawOption struct {
		ID           string            `json:"id"`
		ProviderType string            `json:"provider_type"`
		Type         string            `json:"type"`
		IsActive     bool              `json:"is_active"`
		RenderInfo   PaymentRenderInfo `json:"render_info"`
	}
	var raw rawOption
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	providerType := strings.TrimSpace(raw.ProviderType)
	if providerType == "" {
		providerType = strings.TrimSpace(raw.Type)
	}
	*c = PaymentOptionConfig{
		ID:           strings.TrimSpace(raw.ID),
		ProviderType: providerType,
		IsActive:     raw.IsActive,
		RenderInfo:   raw.RenderInfo,
	}
	return nil
}

// Config is the JSON document used to load configured payment options.
type Config struct {
	Version   string                `json:"version"`
	DefaultID string                `json:"default_id,omitempty"`
	Options   []PaymentOptionConfig `json:"options,omitempty"`
}

// UnmarshalJSON accepts both `options` and legacy `methods` arrays, and `default_id` or `defaultId`.
func (c *Config) UnmarshalJSON(data []byte) error {
	type rawConfig struct {
		Version      string                `json:"version"`
		DefaultID    string                `json:"default_id"`
		DefaultIDAlt string                `json:"defaultId"`
		Options      []PaymentOptionConfig `json:"options"`
		Methods      []PaymentOptionConfig `json:"methods"`
	}
	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	options := make([]PaymentOptionConfig, 0, len(raw.Options)+len(raw.Methods))
	options = append(options, raw.Options...)
	options = append(options, raw.Methods...)
	defaultID := strings.TrimSpace(raw.DefaultID)
	if defaultID == "" {
		defaultID = strings.TrimSpace(raw.DefaultIDAlt)
	}
	*c = Config{
		Version:   strings.TrimSpace(raw.Version),
		DefaultID: defaultID,
		Options:   options,
	}
	return nil
}

// Validate performs structural validation that is independent of provider factory registration.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if len(c.Options) == 0 {
		return fmt.Errorf("at least one payment option is required")
	}

	seenIDs := make(map[string]struct{}, len(c.Options))
	activeCount := 0
	for i, opt := range c.Options {
		id := strings.TrimSpace(opt.ID)
		if id == "" {
			return fmt.Errorf("payment option at index %d is missing id", i)
		}
		if _, ok := seenIDs[id]; ok {
			return fmt.Errorf("duplicate payment option id: %s", id)
		}
		seenIDs[id] = struct{}{}

		if strings.TrimSpace(opt.ProviderType) == "" {
			return fmt.Errorf("payment option %q is missing provider type", id)
		}
		if strings.TrimSpace(opt.RenderInfo.DisplayName) == "" {
			return fmt.Errorf("payment option %q is missing render_info.display_name", id)
		}
		if strings.TrimSpace(opt.RenderInfo.Description) == "" {
			return fmt.Errorf("payment option %q is missing render_info.description", id)
		}
		if opt.RenderInfo.DisplayOrder < 0 {
			return fmt.Errorf("payment option %q has invalid render_info.display_order", id)
		}
		if opt.IsActive {
			activeCount++
		}
	}

	if activeCount == 0 {
		return fmt.Errorf("at least one enabled payment option is required")
	}

	if c.DefaultID != "" {
		defaultOpt, ok := c.findOptionByID(c.DefaultID)
		if !ok {
			return fmt.Errorf("default payment option %q does not exist", c.DefaultID)
		}
		if !defaultOpt.IsActive {
			return fmt.Errorf("default payment option %q must be enabled", c.DefaultID)
		}
	}

	return nil
}

func (c Config) findOptionByID(id string) (PaymentOptionConfig, bool) {
	for _, opt := range c.Options {
		if opt.ID == id {
			return opt, true
		}
	}
	return PaymentOptionConfig{}, false
}

func (c Config) activeOptions() []PaymentOptionConfig {
	active := make([]PaymentOptionConfig, 0, len(c.Options))
	for _, opt := range c.Options {
		if opt.IsActive {
			active = append(active, opt)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		if active[i].RenderInfo.DisplayOrder == active[j].RenderInfo.DisplayOrder {
			return active[i].ID < active[j].ID
		}
		return active[i].RenderInfo.DisplayOrder < active[j].RenderInfo.DisplayOrder
	})
	return active
}
