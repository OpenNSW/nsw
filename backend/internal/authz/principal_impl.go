package authz

import (
	"slices"

	"github.com/OpenNSW/nsw/internal/auth"
)

// principal is the unexported Principal implementation backed by
// *auth.AuthContext. Scope resolution is delegated to the originating
// Manager so each request can reuse the manager's static configuration
// without copying maps.
type principal struct {
	authCtx *auth.AuthContext
	manager *Manager
}

// Compile-time interface satisfaction check.
var _ Principal = (*principal)(nil)

func (p *principal) Kind() Kind {
	if p.authCtx.User != nil {
		return KindUser
	}
	return KindClient
}

func (p *principal) Subject() string {
	if p.authCtx.User != nil {
		if p.authCtx.User.ID != "" {
			return p.authCtx.User.ID
		}
		return p.authCtx.User.IDPUserID
	}
	if p.authCtx.Client != nil {
		return p.authCtx.Client.ClientID
	}
	return ""
}

func (p *principal) HasScope(scope string) bool {
	if scope == "" {
		return false
	}
	if p.authCtx.User != nil {
		for _, role := range p.authCtx.User.Roles {
			if slices.Contains(p.manager.roleScopes[role], scope) {
				return true
			}
		}
		return false
	}
	if p.authCtx.Client != nil {
		return slices.Contains(p.manager.clientScopes[p.authCtx.Client.ClientID], scope)
	}
	return false
}

func (p *principal) HasRole(role string) bool {
	if p.authCtx.User == nil {
		return false
	}
	return slices.Contains(p.authCtx.User.Roles, role)
}

func (p *principal) User() (*auth.UserContext, bool) {
	if p.authCtx.User == nil {
		return nil, false
	}
	return p.authCtx.User, true
}

func (p *principal) Client() (*auth.ClientContext, bool) {
	if p.authCtx.Client == nil {
		return nil, false
	}
	return p.authCtx.Client, true
}
