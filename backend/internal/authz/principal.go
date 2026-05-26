// Package authz provides authorization primitives that complement the
// authentication layer in internal/auth.
//
// The package introduces a Principal abstraction that unifies the two
// principal types produced by the auth middleware (user vs. client) behind
// a single interface, and a Manager that resolves OAuth2-style scopes from
// the principal's roles (for users) or client_id (for M2M clients) using a
// static configuration.
//
// The package deliberately leaves business policy decisions to the
// downstream service: callers receive a Principal, branch on Kind, and use
// HasRole/HasScope to make decisions. Coarse-grained "is this scope present
// at all?" gating is provided by Manager.RequireScope as HTTP middleware.
package authz

import (
	"errors"

	"github.com/OpenNSW/nsw/internal/auth"
)

// Kind discriminates between principal types so callers can branch on
// whether the caller is a human user or an M2M client.
type Kind int

const (
	// KindUser indicates the principal was authenticated via the OAuth2
	// authorization_code grant (a human user with identity claims and roles).
	KindUser Kind = iota + 1
	// KindClient indicates the principal was authenticated via the OAuth2
	// client_credentials grant (an M2M client identified by client_id only).
	KindClient
)

// Principal is the unified view of an authenticated caller. It wraps the
// existing *auth.AuthContext and exposes authorization-friendly helpers
// without coupling callers to the underlying user/client structs.
//
// All methods are safe to call on any Principal returned by Manager.Principal.
// HasRole returns false for client principals; Client returns ok=false for
// user principals (and vice versa).
type Principal interface {
	// Kind returns whether the principal is a user or a client.
	Kind() Kind
	// Subject returns a stable identifier for the principal: the persisted
	// user ID (falling back to the IdP subject if not persisted yet) for
	// users, or the client_id for M2M clients.
	Subject() string
	// HasScope reports whether the principal has the given scope, as
	// resolved by the Manager that produced this Principal (role-map for
	// users, client-allowlist for clients).
	HasScope(scope string) bool
	// HasRole reports whether the principal carries the given role claim.
	// Always returns false for client principals.
	HasRole(role string) bool
	// User returns the underlying user context; ok=false for clients.
	User() (*auth.UserContext, bool)
	// Client returns the underlying client context; ok=false for users.
	Client() (*auth.ClientContext, bool)
}

// Sentinel errors returned by services that perform downstream
// authorization checks. Handlers use errors.Is to map these onto HTTP
// status codes.
var (
	// ErrUnauthenticated indicates no principal is available on the request.
	// Handlers should map this to 401 Unauthorized.
	ErrUnauthenticated = errors.New("authz: unauthenticated")
	// ErrForbidden indicates the principal is authenticated but not allowed
	// to perform the requested action on the requested resource.
	// Handlers should map this to 403 Forbidden.
	ErrForbidden = errors.New("authz: forbidden")
)
