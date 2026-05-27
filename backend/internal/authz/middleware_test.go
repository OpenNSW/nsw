package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenNSW/nsw/internal/auth"
)

func newNextHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireScope_NilManagerPanicsAtConstruction(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when RequireScope is called on a nil Manager")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil Manager") {
			t.Fatalf("expected panic message mentioning nil Manager, got %v (%T)", r, r)
		}
	}()
	var mgr *Manager
	_ = mgr.RequireScope("anything")
}

func TestRequireScope_AnonymousReturns401(t *testing.T) {
	mgr := newTestManager(t)

	var nextCalled bool
	wrapped := mgr.RequireScope("consignments:read")(newNextHandler(&nextCalled))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"unauthorized"`) {
		t.Fatalf("expected JSON unauthorized body, got %q", rec.Body.String())
	}
	if nextCalled {
		t.Fatal("expected next handler NOT to be called for anonymous request")
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}
}

func TestRequireScope_UserMissingScopeReturns403(t *testing.T) {
	mgr := newTestManager(t)
	// cha has consignments:read but not consignments:write
	ctx := context.WithValue(context.Background(), auth.AuthContextKey,
		&auth.AuthContext{User: &auth.UserContext{IDPUserID: "u", Roles: []string{"cha"}}})

	var nextCalled bool
	wrapped := mgr.RequireScope("consignments:write")(newNextHandler(&nextCalled))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"forbidden"`) {
		t.Fatalf("expected JSON forbidden body, got %q", rec.Body.String())
	}
	if nextCalled {
		t.Fatal("expected next handler NOT to be called when scope is missing")
	}
}

func TestRequireScope_UserWithScopeReachesHandler(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.WithValue(context.Background(), auth.AuthContextKey,
		&auth.AuthContext{User: &auth.UserContext{IDPUserID: "u", Roles: []string{"trader"}}})

	var nextCalled bool
	wrapped := mgr.RequireScope("consignments:write")(newNextHandler(&nextCalled))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called when scope is present")
	}
}

func TestRequireScope_ClientWithScopeReachesHandler(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.WithValue(context.Background(), auth.AuthContextKey,
		&auth.AuthContext{Client: &auth.ClientContext{ClientID: "LANKAPAY_M2M"}})

	var nextCalled bool
	wrapped := mgr.RequireScope("payments:webhook")(newNextHandler(&nextCalled))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called when client has scope")
	}
}

func TestRequireScope_ClientMissingScopeReturns403(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.WithValue(context.Background(), auth.AuthContextKey,
		&auth.AuthContext{Client: &auth.ClientContext{ClientID: "STRANGER"}})

	var nextCalled bool
	wrapped := mgr.RequireScope("payments:webhook")(newNextHandler(&nextCalled))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if nextCalled {
		t.Fatal("expected next handler NOT to be called for unmapped client")
	}
}
