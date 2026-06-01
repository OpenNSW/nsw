package paymentsv2

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/OpenNSW/nsw/backend/internal/paymentsv2/gateways"
)

// GatewayRegistry manages the discovery and lookup of payment gateways.
type GatewayRegistry interface {
	// Get retrieves a gateway implementation by its ID.
	Get(id string) (gateways.PaymentGateway, error)

	// ListInfo returns the aggregated metadata for all supported gateways.
	ListInfo() []GatewayInfo
}

type paymentRegistry struct {
	mu       sync.RWMutex
	gateways map[string]gateways.PaymentGateway
	infos    map[string]GatewayInfo
}

// NewRegistry initializes a new registry by loading configuration from a file.
// It maps gateway IDs from the config to the provided map of implementations.
func NewRegistry(configPath string, gateways map[string]gateways.PaymentGateway) (GatewayRegistry, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment methods config: %w", err)
	}

	var config struct {
		Version string        `json:"version"`
		Methods []GatewayInfo `json:"methods"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment methods config: %w", err)
	}

	registry := &paymentRegistry{
		gateways: gateways,
		infos:    make(map[string]GatewayInfo),
	}

	for _, info := range config.Methods {
		registry.infos[info.ID] = info

		// If a gateway implementation exists for this info, apply its config
		if gateway, ok := gateways[info.ID]; ok {
			if len(info.Config) > 0 {
				if err := gateway.ApplyConfig(info.Config); err != nil {
					return nil, fmt.Errorf("failed to apply config for gateway %s: %w", info.ID, err)
				}
			}
		}
	}

	return registry, nil
}

func (r *paymentRegistry) Get(id string) (gateways.PaymentGateway, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gateway, ok := r.gateways[id]
	if !ok {
		return nil, fmt.Errorf("gateway %s not found in registry", id)
	}

	return gateway, nil
}

func (r *paymentRegistry) ListInfo() []GatewayInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activeMethods []GatewayInfo
	for _, info := range r.infos {
		if info.IsActive {
			// Sanitize: Return only UI-safe fields
			activeMethods = append(activeMethods, GatewayInfo{
				ID:         info.ID,
				IsActive:   info.IsActive,
				RenderInfo: info.RenderInfo,
				// Config is omitted intentionally
			})
		}
	}

	// Sort by DisplayOrder for consistent UI presentation
	sort.Slice(activeMethods, func(i, j int) bool {
		return activeMethods[i].RenderInfo.DisplayOrder < activeMethods[j].RenderInfo.DisplayOrder
	})

	return activeMethods
}
