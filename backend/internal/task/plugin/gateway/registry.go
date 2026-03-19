package gateway

import (
	"fmt"
	"sync"
)

// Registry manages the set of available payment gateways
type Registry struct {
	mu       sync.RWMutex
	gateways map[string]PaymentGateway
}

// NewRegistry creates a new empty Registry
func NewRegistry() *Registry {
	return &Registry{
		gateways: make(map[string]PaymentGateway),
	}
}

// Register adds a gateway to the registry
func (r *Registry) Register(gw PaymentGateway) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gateways[gw.ID()] = gw
}

// Get retrieves a gateway by its ID
func (r *Registry) Get(id string) (PaymentGateway, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	gw, exists := r.gateways[id]
	if !exists {
		return nil, fmt.Errorf("payment gateway not found: %s", id)
	}
	return gw, nil
}
