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

// Configured loads enabled payment options from JSON and resolves providers via configured instances.
type Configured struct {
	mu              sync.RWMutex
	providersByType map[string]PaymentProvider
	providers       map[string]PaymentProvider
	infos           map[string]PaymentProviderInfo
	types           map[string][]string
	defaultID       string
	// cached sorted infos populated during LoadConfig to avoid recomputing on every ListInfo call
	sortedInfos []PaymentProviderInfo
}

// NewConfigured creates an empty registry with no providers registered.
func NewConfigured() *Configured {
	return &Configured{
		providersByType: make(map[string]PaymentProvider),
		providers:       make(map[string]PaymentProvider),
		infos:           make(map[string]PaymentProviderInfo),
		types:           make(map[string][]string),
	}
}

// RegisterProvider adds a provider implementation keyed by configured provider type.
func (r *Configured) RegisterProvider(providerType string, provider PaymentProvider) {
	if r == nil {
		return
	}
	providerType = strings.TrimSpace(providerType)
	if providerType == "" || provider == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providersByType[providerType] = provider
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
	providersByType := make(map[string]PaymentProvider, len(r.providersByType))
	for k, v := range r.providersByType {
		providersByType[k] = v
	}
	r.mu.RUnlock()

	for _, opt := range active {
		provider, ok := providersByType[opt.ProviderType]
		if !ok {
			return fmt.Errorf("unsupported provider type %q for enabled payment option %q", opt.ProviderType, opt.ID)
		}
		if provider == nil {
			return fmt.Errorf("provider for type %q is nil", opt.ProviderType)
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
	// build and cache sorted infos slice
	sorted := make([]PaymentProviderInfo, 0, len(infos))
	for _, info := range infos {
		sorted = append(sorted, info)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].RenderInfo.DisplayOrder == sorted[j].RenderInfo.DisplayOrder {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].RenderInfo.DisplayOrder < sorted[j].RenderInfo.DisplayOrder
	})
	r.sortedInfos = sorted
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
	// return a copy of the cached sorted infos to avoid mutation by caller
	out := make([]PaymentProviderInfo, len(r.sortedInfos))
	copy(out, r.sortedInfos)
	return out
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

// NewFromFile is a convenience helper for loading a registry from JSON with registered providers.
func NewFromFile(path string, providersByType map[string]PaymentProvider) (*Configured, error) {
	reg := NewConfigured()
	for providerType, provider := range providersByType {
		reg.RegisterProvider(providerType, provider)
	}
	if err := reg.LoadFromFile(path); err != nil {
		return nil, err
	}
	return reg, nil
}
