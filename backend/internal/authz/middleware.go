package authz

import (
	"log/slog"
	"net/http"
)

// RequireScope returns a middleware that rejects requests whose principal
// lacks the given scope. Use it to declare a coarse authorization gate per
// route at the application's router setup.
//
// Behaviour:
//   - No principal on the request (anonymous): 401 Unauthorized.
//   - Principal lacks the scope: 403 Forbidden.
//   - Otherwise: next handler is invoked unchanged.
//
// The JSON response shape matches the existing auth middleware so clients
// see a consistent error format across authentication and authorization
// failures.
//
// Panics at construction time if the receiver is nil. Failing fast at
// route-wiring time produces a clear bootstrap error instead of a
// closure that panics on the first request.
func (m *Manager) RequireScope(scope string) func(http.Handler) http.Handler {
	if m == nil {
		panic("authz: RequireScope called on a nil Manager")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := m.Principal(r.Context())
			if !ok {
				slog.Debug("authz: no principal on request", "scope", scope, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"authentication required"}`))
				return
			}
			if !p.HasScope(scope) {
				slog.Warn("authz: principal lacks required scope",
					"scope", scope,
					"kind", p.Kind(),
					"subject", p.Subject(),
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden","message":"insufficient scope"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
