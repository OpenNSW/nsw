package authz

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/auth"
)

// Manager owns the static authorization configuration and exposes helpers
// for building Principals and constructing scope-checking middleware. The
// manager mirrors the pattern of auth.Manager: construct once at startup
// and inject the same instance into handlers and services that need it.
type Manager struct {
	roleScopes   map[string][]string
	clientScopes map[string][]string
}

// NewManager constructs a Manager from the given configuration. Returns an
// error if the configuration is structurally malformed. An empty config is
// permitted; the resulting manager treats every scope check as false.
func NewManager(cfg Config) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authz config: %w", err)
	}
	return &Manager{
		roleScopes:   cloneScopeMap(cfg.RoleScopes),
		clientScopes: cloneScopeMap(cfg.ClientScopes),
	}, nil
}

// Principal returns the authenticated principal for the given request
// context, resolved from the auth context injected by auth.Middleware.
// ok=false when no auth context is attached (anonymous request) or when
// the attached context is empty (neither user nor client present).
func (m *Manager) Principal(ctx context.Context) (Principal, bool) {
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil {
		return nil, false
	}
	if authCtx.User == nil && authCtx.Client == nil {
		return nil, false
	}
	return &principal{authCtx: authCtx, manager: m}, true
}

func cloneScopeMap(src map[string][]string) map[string][]string {
	if len(src) == 0 {
		return map[string][]string{}
	}
	dst := make(map[string][]string, len(src))
	for k, v := range src {
		copied := make([]string, len(v))
		copy(copied, v)
		dst[k] = copied
	}
	return dst
}
