package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

// Configured loads enabled payment options from JSON and resolves providers via factories.
type Configured struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
	providers map[string]PaymentProvider
	infos     map[string]PaymentProviderInfo
	types     map[string][]string
	defaultID string
}

// NewConfigured creates an empty registry with no factories registered.
func NewConfigured() *Configured {
	return &Configured{
		factories: make(map[string]ProviderFactory),
		providers: make(map[string]PaymentProvider),
		infos:     make(map[string]PaymentProviderInfo),
		types:     make(map[string][]string),
	}
}

// RegisterFactory adds a provider factory keyed by configured provider type.
func (r *Configured) RegisterFactory(providerType string, factory ProviderFactory) {
	if r == nil {
		return
	}
	providerType = strings.TrimSpace(providerType)
	if providerType == "" || factory == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[providerType] = factory
}

// LoadFromFile loads a registry configuration from a JSON file path and instantiates all enabled providers.
func (r *Configured) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open payment registry config: %w", err)
	}
	defer func() { _ = file.Close() }()
	return r.LoadFromReader(file)
}

// LoadFromReader loads a registry configuration from an arbitrary reader.
func (r *Configured) LoadFromReader(reader io.Reader) error {
	var cfg Config
	if err := json.NewDecoder(reader).Decode(&cfg); err != nil {
		return fmt.Errorf("decode payment registry config: %w", err)
	}
	return r.LoadConfig(cfg)
}

// LoadConfig validates the configuration and prepares the runtime registry state.
func (r *Configured) LoadConfig(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	active := cfg.activeOptions()
	providers := make(map[string]PaymentProvider, len(active))
	infos := make(map[string]PaymentProviderInfo, len(active))
	types := make(map[string][]string)

	r.mu.RLock()
	factories := make(map[string]ProviderFactory, len(r.factories))
	for k, v := range r.factories {
		factories[k] = v
	}
	r.mu.RUnlock()

	for _, opt := range active {
		factory, ok := factories[opt.ProviderType]
		if !ok {
			return fmt.Errorf("unsupported provider type %q for enabled payment option %q", opt.ProviderType, opt.ID)
		}
		provider := factory()
		if provider == nil {
			return fmt.Errorf("provider factory for type %q returned nil", opt.ProviderType)
		}
		providers[opt.ID] = provider
		infos[opt.ID] = PaymentProviderInfo{
			ID:           opt.ID,
			ProviderType: opt.ProviderType,
			IsActive:     true,
			RenderInfo:   opt.RenderInfo,
		}
		types[opt.ProviderType] = append(types[opt.ProviderType], opt.ID)
	}

	defaultID := cfg.DefaultID
	if defaultID == "" {
		defaultID = active[0].ID
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = providers
	r.infos = infos
	r.types = types
	r.defaultID = defaultID
	return nil
}

// Get retrieves a provider implementation by its configured option ID.
func (r *Configured) Get(id string) (PaymentProvider, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.providers[id]; ok {
		return p, nil
	}
	if _, ok := r.infos[id]; ok {
		return nil, fmt.Errorf("payment option %q is not enabled", id)
	}
	return nil, fmt.Errorf("payment option not found: %s", id)
}

// GetByType retrieves a provider implementation by configured provider type.
func (r *Configured) GetByType(providerType string) (PaymentProvider, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	providerType = strings.TrimSpace(providerType)
	if providerType == "" {
		return nil, fmt.Errorf("provider type is required")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.types[providerType]
	if len(ids) == 0 {
		return nil, fmt.Errorf("payment provider type not found: %s", providerType)
	}
	if r.defaultID != "" {
		if info, ok := r.infos[r.defaultID]; ok && info.ProviderType == providerType {
			return r.providers[r.defaultID], nil
		}
	}
	return r.providers[ids[0]], nil
}

// ListInfo returns metadata for all enabled payment options sorted by display order and ID.
func (r *Configured) ListInfo() []PaymentProviderInfo {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	infos := make([]PaymentProviderInfo, 0, len(r.infos))
	for _, info := range r.infos {
		infos = append(infos, info)
	}
	sort.SliceStable(infos, func(i, j int) bool {
		if infos[i].RenderInfo.DisplayOrder == infos[j].RenderInfo.DisplayOrder {
			return infos[i].ID < infos[j].ID
		}
		return infos[i].RenderInfo.DisplayOrder < infos[j].RenderInfo.DisplayOrder
	})
	return infos
}

// GetDefault returns the primary provider implementation resolved during config loading.
func (r *Configured) GetDefault() (PaymentProvider, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	r.mu.RLock()
	defaultID := r.defaultID
	r.mu.RUnlock()
	if defaultID == "" {
		return nil, fmt.Errorf("no default payment provider configured")
	}
	return r.Get(defaultID)
}

// NewFromFile is a convenience helper for loading a registry from JSON with registered factories.
func NewFromFile(path string, factories map[string]ProviderFactory) (*Configured, error) {
	reg := NewConfigured()
	for providerType, factory := range factories {
		reg.RegisterFactory(providerType, factory)
	}
	if err := reg.LoadFromFile(path); err != nil {
		return nil, err
	}
	return reg, nil
}
