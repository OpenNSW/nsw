package authz

import (
	"context"
	"testing"

	"github.com/OpenNSW/nsw/internal/auth"
)

// contextWithUser builds a request context carrying a UserContext, matching
// what auth.Middleware would inject for a JWT issued via authorization_code.
func contextWithUser(t *testing.T, user *auth.UserContext) context.Context {
	t.Helper()
	return context.WithValue(context.Background(), auth.AuthContextKey, &auth.AuthContext{User: user})
}

// contextWithClient builds a request context carrying a ClientContext, matching
// what auth.Middleware would inject for a JWT issued via client_credentials.
func contextWithClient(t *testing.T, client *auth.ClientContext) context.Context {
	t.Helper()
	return context.WithValue(context.Background(), auth.AuthContextKey, &auth.AuthContext{Client: client})
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(Config{
		RoleScopes: map[string][]string{
			"trader": {"consignments:read", "consignments:write"},
			"cha":    {"consignments:read"},
		},
		ClientScopes: map[string][]string{
			"LANKAPAY_M2M": {"payments:webhook"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error from NewManager: %v", err)
	}
	return mgr
}

func TestNewManager_EmptyConfigIsValid(t *testing.T) {
	mgr, err := NewManager(Config{})
	if err != nil {
		t.Fatalf("expected nil error for empty config, got %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNewManager_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "empty scope in role", cfg: Config{RoleScopes: map[string][]string{"trader": {""}}}},
		{name: "empty role name", cfg: Config{RoleScopes: map[string][]string{"": {"consignments:read"}}}},
		{name: "empty scope in client", cfg: Config{ClientScopes: map[string][]string{"X": {""}}}},
		{name: "empty client id", cfg: Config{ClientScopes: map[string][]string{"": {"payments:webhook"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewManager(tt.cfg); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestManagerPrincipal_NoAuthContext(t *testing.T) {
	mgr := newTestManager(t)
	if _, ok := mgr.Principal(context.Background()); ok {
		t.Fatal("expected ok=false when no auth context attached")
	}
}

func TestManagerPrincipal_EmptyAuthContext(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.WithValue(context.Background(), auth.AuthContextKey, &auth.AuthContext{})
	if _, ok := mgr.Principal(ctx); ok {
		t.Fatal("expected ok=false when both user and client are nil")
	}
}

func TestPrincipal_UserKindAndAccessors(t *testing.T) {
	mgr := newTestManager(t)
	user := &auth.UserContext{
		ID:        "USER-DB-001",
		IDPUserID: "USER-IDP-001",
		Email:     "trader@example.com",
		OUHandle:  "ou-trader",
		Roles:     []string{"trader"},
	}
	p, ok := mgr.Principal(contextWithUser(t, user))
	if !ok {
		t.Fatal("expected ok=true for user context")
	}
	if p.Kind() != KindUser {
		t.Fatalf("expected KindUser, got %d", p.Kind())
	}
	if p.Subject() != "USER-DB-001" {
		t.Fatalf("expected persisted ID as subject, got %q", p.Subject())
	}
	gotUser, gotOK := p.User()
	if !gotOK || gotUser != user {
		t.Fatalf("expected User() to return wrapped user, got %v (ok=%v)", gotUser, gotOK)
	}
	if _, clientOK := p.Client(); clientOK {
		t.Fatal("expected Client() ok=false for user principal")
	}
}

func TestPrincipal_SubjectFallsBackToIDPUserID(t *testing.T) {
	mgr := newTestManager(t)
	user := &auth.UserContext{IDPUserID: "USER-IDP-002", Roles: []string{"trader"}}
	p, _ := mgr.Principal(contextWithUser(t, user))
	if p.Subject() != "USER-IDP-002" {
		t.Fatalf("expected IDPUserID as subject when ID is empty, got %q", p.Subject())
	}
}

func TestPrincipal_HasRole(t *testing.T) {
	mgr := newTestManager(t)
	user := &auth.UserContext{IDPUserID: "u", Roles: []string{"trader", "ops"}}
	p, _ := mgr.Principal(contextWithUser(t, user))
	if !p.HasRole("trader") {
		t.Error("expected HasRole(trader)=true")
	}
	if !p.HasRole("ops") {
		t.Error("expected HasRole(ops)=true")
	}
	if p.HasRole("cha") {
		t.Error("expected HasRole(cha)=false")
	}
}

func TestPrincipal_HasRole_FalseForClient(t *testing.T) {
	mgr := newTestManager(t)
	p, _ := mgr.Principal(contextWithClient(t, &auth.ClientContext{ClientID: "LANKAPAY_M2M"}))
	if p.HasRole("trader") {
		t.Error("expected HasRole=false for client principals")
	}
}

func TestPrincipal_HasScope_UserViaRoleMap(t *testing.T) {
	mgr := newTestManager(t)
	tests := []struct {
		name  string
		roles []string
		scope string
		want  bool
	}{
		{name: "trader has read", roles: []string{"trader"}, scope: "consignments:read", want: true},
		{name: "trader has write", roles: []string{"trader"}, scope: "consignments:write", want: true},
		{name: "cha has read", roles: []string{"cha"}, scope: "consignments:read", want: true},
		{name: "cha lacks write", roles: []string{"cha"}, scope: "consignments:write", want: false},
		{name: "multi-role union", roles: []string{"cha", "trader"}, scope: "consignments:write", want: true},
		{name: "unknown role grants nothing", roles: []string{"ghost"}, scope: "consignments:read", want: false},
		{name: "no roles grants nothing", roles: nil, scope: "consignments:read", want: false},
		{name: "empty scope query rejected", roles: []string{"trader"}, scope: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &auth.UserContext{IDPUserID: "u", Roles: tt.roles}
			p, _ := mgr.Principal(contextWithUser(t, user))
			if got := p.HasScope(tt.scope); got != tt.want {
				t.Fatalf("HasScope(%q) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

func TestPrincipal_ClientKindAndScope(t *testing.T) {
	mgr := newTestManager(t)
	p, ok := mgr.Principal(contextWithClient(t, &auth.ClientContext{ClientID: "LANKAPAY_M2M"}))
	if !ok {
		t.Fatal("expected ok=true for client context")
	}
	if p.Kind() != KindClient {
		t.Fatalf("expected KindClient, got %d", p.Kind())
	}
	if p.Subject() != "LANKAPAY_M2M" {
		t.Fatalf("expected client_id as subject, got %q", p.Subject())
	}
	if !p.HasScope("payments:webhook") {
		t.Error("expected client to have payments:webhook scope")
	}
	if p.HasScope("consignments:read") {
		t.Error("expected client to NOT have consignments:read scope")
	}
	gotClient, gotOK := p.Client()
	if !gotOK || gotClient.ClientID != "LANKAPAY_M2M" {
		t.Fatalf("expected Client() to return wrapped client, got %v (ok=%v)", gotClient, gotOK)
	}
	if _, userOK := p.User(); userOK {
		t.Fatal("expected User() ok=false for client principal")
	}
}

func TestPrincipal_UnknownClientHasNoScopes(t *testing.T) {
	mgr := newTestManager(t)
	p, _ := mgr.Principal(contextWithClient(t, &auth.ClientContext{ClientID: "STRANGER"}))
	if p.HasScope("payments:webhook") {
		t.Error("expected unmapped client to have no scopes")
	}
}

func TestManagerPrincipal_DoesNotMutateInputMaps(t *testing.T) {
	// Ensure cloneScopeMap actually decouples the manager from caller-owned maps.
	role := map[string][]string{"trader": {"consignments:read"}}
	mgr, err := NewManager(Config{RoleScopes: role})
	if err != nil {
		t.Fatalf("unexpected NewManager error: %v", err)
	}
	role["trader"] = append(role["trader"], "consignments:write")

	user := &auth.UserContext{IDPUserID: "u", Roles: []string{"trader"}}
	p, _ := mgr.Principal(contextWithUser(t, user))
	if p.HasScope("consignments:write") {
		t.Fatal("expected manager to be isolated from post-construction mutation of input maps")
	}
}
