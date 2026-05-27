# authz

Authorization primitives for the NSW backend. Complements
[`internal/auth`](../auth/) — `auth` answers *"who is the caller?"*, this
package answers *"what is the caller allowed to do?"*.

The package is designed around a single rule:
**the downstream service decides who is allowed.** Middleware is a coarse
gate; handlers are dumb HTTP adapters; services own the policy.

---

## Contents

- `Principal` — a unified view of an authenticated caller (user *or*
  M2M client) with `HasRole` / `HasScope` helpers.
- `Manager` — owns the static scope configuration and constructs
  `Principal`s and middleware.
- `RequireScope(scope)` — per-route HTTP middleware factory that
  rejects callers lacking a declared scope.
- `ErrUnauthenticated` / `ErrForbidden` — sentinel errors that services
  return and handlers map onto 401 / 403.

---

## The layered model

| Layer | What it decides | Where it lives |
|---|---|---|
| `auth.Middleware` (existing) | Is the JWT valid? Who is the caller? | `internal/auth/middleware.go` |
| `authz.RequireScope` | Does the caller carry *any* scope strong enough to even attempt this route? | route registration in `bootstrap/app.go` |
| Handler | HTTP wiring only. Pulls the `Principal` from context, forwards it to the service, maps `ErrForbidden` → 403. | per-feature `router.go` |
| **Service** | All fine-grained decisions: ownership ("is this consignment owned by this user's company?"), role branching, M2M scope checks. | per-feature `service.go` |

> Rule of thumb: if the answer depends on a database lookup, it belongs in
> the service. If the answer is "is this scope string in the principal's
> set?", it belongs in middleware.

---

## Wiring it up

`Manager` is constructed once at startup and injected into the handlers
that need it, mirroring the existing `auth.Manager` pattern.

```go
// internal/app/bootstrap/app.go
authzManager, err := authz.NewManager(authz.Config{
    RoleScopes: map[string][]string{
        "trader": {"consignments:read", "consignments:write"},
        "cha":    {"consignments:read", "consignments:initialize"},
    },
    ClientScopes: map[string][]string{
        "LANKAPAY_M2M":  {"payments:webhook"},
        "FCAU_TO_NSW":   {"consignments:read:all"},
    },
})
if err != nil {
    return fmt.Errorf("initialize authz manager: %w", err)
}

withAuth   := authManager.Middleware()
withScope  := authzManager.RequireScope            // closure-friendly alias

mux.Handle("GET /api/v1/consignments",
    withAuth(withScope("consignments:read")(
        http.HandlerFunc(consignmentRouter.HandleGetConsignments))))
```

`auth.Middleware` must run *before* `RequireScope`, since the latter reads
the auth context the former injects. Empty `Config` is valid — the
resulting manager treats every `HasScope` as `false`, useful while you're
still wiring scopes for a feature.

---

## Migrating a handler — the v1 → v2 pattern

Here is the before/after for a typical handler. The "before" is what is
in `internal/consignment/router.go` today; the "after" is what v2 will
look like.

### Before (today)

```go
// router.go
func (c *Router) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
    authCtx := auth.GetAuthContext(r.Context())
    if authCtx == nil || authCtx.User == nil {       // boilerplate; rejects M2M
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    // TODO: Proper AuthZ need to be implemented.
    role := r.URL.Query().Get("role")                // user-supplied role — critical flaw
    if role == "" { role = "trader" }
    // ... look up company by OUHandle, filter, etc.
}
```

### After (with this package)

**1. Route declares its scope (coarse gate):**

```go
// bootstrap/app.go
mux.Handle("GET /api/v1/consignments",
    withAuth(withScope("consignments:read")(
        http.HandlerFunc(consignmentRouter.HandleGetConsignments))))
```

**2. Handler shrinks to "extract principal, forward to service":**

```go
// router.go
func (c *Router) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
    p, ok := c.authz.Principal(r.Context())
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    out, err := c.cs.ListConsignmentsFor(r.Context(), p, parseFilter(r))
    switch {
    case errors.Is(err, authz.ErrForbidden):
        http.Error(w, "Forbidden", http.StatusForbidden)
    case err != nil:
        slog.Error("list consignments", "error", err)
        http.Error(w, "internal", http.StatusInternalServerError)
    default:
        writeJSON(w, http.StatusOK, out)
    }
}
```

**3. Service owns the policy:**

```go
// service.go
func (s *Service) ListConsignmentsFor(ctx context.Context, p authz.Principal, f Filter) ([]Consignment, error) {
    switch p.Kind() {
    case authz.KindUser:
        u, _ := p.User()
        co, err := s.companyService.GetCompanyByOUHandle(ctx, u.OUHandle)
        if err != nil {
            return nil, authz.ErrForbidden    // unknown company == not allowed
        }
        switch {
        case p.HasRole("trader"):
            f.TraderCompanyID = &co.ID
        case p.HasRole("cha"):
            f.CHACompanyID = &co.ID
        default:
            return nil, authz.ErrForbidden
        }
    case authz.KindClient:
        if !p.HasScope("consignments:read:all") {
            return nil, authz.ErrForbidden
        }
        // M2M with the right scope sees everything — no company filter
    }
    return s.list(ctx, f)
}
```

What changed:

- The `role=` query parameter is **gone**. Role comes from the JWT roles
  claim via `p.HasRole(...)`.
- M2M clients can now reach this endpoint — the service handles them
  with a different policy (scope-gated, no company filter).
- The handler has *no* business logic. It maps HTTP to/from the service.
- The route's `consignments:read` scope is the coarse gate. The
  service's `HasRole` / `HasScope` checks are the fine gate.

---

## Patterns by access shape

### Frontend-only endpoint

A scope only mapped to user roles, never to a client. M2M tokens will be
rejected at the middleware (`RequireScope`) with 403.

```go
RoleScopes:   {"trader": {"profile:write"}}
ClientScopes: {/* no client has profile:write */}
```

```go
mux.Handle("PUT /api/v1/profile",
    withAuth(withScope("profile:write")(handler)))
```

### M2M-only endpoint (webhook, internal service)

A scope only mapped to specific client IDs.

```go
RoleScopes:   {/* no role has payments:webhook */}
ClientScopes: {"LANKAPAY_M2M": {"payments:webhook"}}
```

```go
mux.Handle("POST /api/v1/payments/webhook",
    withAuth(withScope("payments:webhook")(handler)))
```

### Mixed endpoint (same route, different policies)

Same scope appears in both maps. The middleware lets both through; the
service branches on `Kind`.

```go
RoleScopes:   {"trader": {"consignments:read"}, "cha": {"consignments:read"}}
ClientScopes: {"FCAU_TO_NSW": {"consignments:read"}}
```

The service then decides whether the user sees only their company's data
or whether the M2M sees everything (see the migration example above).

---

## Configuration model

Scopes are *derived* in this package, not carried in the JWT. There are
two maps:

```go
type Config struct {
    RoleScopes   map[string][]string  // for OAuth2 authorization_code (users)
    ClientScopes map[string][]string  // for OAuth2 client_credentials (M2M)
}
```

- **`RoleScopes`** maps a JWT `roles[]` value to the scopes that role
  grants. A user with multiple roles gets the **union** of their roles'
  scopes.
- **`ClientScopes`** maps an OAuth2 `client_id` to the scopes that
  client is allowed to use. M2M scopes are looked up by `client_id` only,
  not by any role claim (M2M tokens don't carry roles).

`Manager` clones both maps at construction, so post-startup mutation of
the input does **not** leak into the manager.

### Adding a new scope

1. Pick a string. Convention: `resource:action` (e.g.
   `consignments:write`) or `resource:action:qualifier` (e.g.
   `consignments:read:all` for "M2M sees everything").
2. Add it to the relevant map(s) in `Config`.
3. Declare it on the route(s) via `withScope("...")`.
4. If the policy is fine-grained, also check it (or `HasRole`) inside
   the service.

That's it. There's no central scope registry yet — scope strings are
literal in `bootstrap/app.go` and `service.go`. If you grow many scopes,
collect them as `const` declarations in this package later.

---

## Testing patterns

The unit tests in this package construct `Principal`s directly by
injecting an `auth.AuthContext` into a `context.Context` — no JWT
signing required. Copy this pattern when testing services that take a
`Principal`:

```go
func ctxWithUser(roles ...string) context.Context {
    return context.WithValue(context.Background(), auth.AuthContextKey,
        &auth.AuthContext{User: &auth.UserContext{
            ID:       "user-1",
            OUHandle: "ou-test",
            Roles:    roles,
        }})
}

func ctxWithClient(clientID string) context.Context {
    return context.WithValue(context.Background(), auth.AuthContextKey,
        &auth.AuthContext{Client: &auth.ClientContext{ClientID: clientID}})
}

func TestListConsignments_TraderSeesOnlyOwnCompany(t *testing.T) {
    mgr, _ := authz.NewManager(authz.Config{
        RoleScopes: map[string][]string{"trader": {"consignments:read"}},
    })
    p, _ := mgr.Principal(ctxWithUser("trader"))

    out, err := svc.ListConsignmentsFor(ctx, p, Filter{})
    // ... assert filter applied
}
```

For middleware-level tests, use `httptest.NewRecorder` and
`req.WithContext(ctx)`. See `middleware_test.go` in this package for the
full pattern.

---

## Common mistakes

- **Reading `auth.AuthContext.User.Roles` directly in handlers.** Use
  `p.HasRole(...)` — handlers should not know about the underlying auth
  context shape.
- **Putting ownership checks in middleware.** Middleware shouldn't hit
  the database. Anything that requires a lookup of "does this resource
  belong to this caller?" goes in the service.
- **Returning a raw error from a service when it means "forbidden".**
  Wrap with `%w` from `authz.ErrForbidden` (or return the sentinel
  directly) so the handler can map it to 403 via `errors.Is`.
- **Forgetting that `RequireScope` returns 401 for anonymous and 403
  for missing scope.** That's the conventional distinction; don't
  collapse them.
- **Configuring an empty `Config` and wondering why all routes 403.**
  Empty maps mean nobody has any scope. Either populate the config or
  don't attach `RequireScope` yet.

---

## What's NOT in this package (yet)

These are deliberately deferred. The interfaces are seams — when any of
these become real, callers don't change.

- **JWT-issued scope claims.** Today scopes are derived from static
  config. When Thunder (the IdP) starts issuing `scope`/`scp` claims,
  swap the resolver inside `principal_impl.go`; `HasScope` semantics
  are unchanged.
- **Role hierarchy.** Per the org's group/role model, roles live at the
  private-sector OU and are inherited by child OUs. Today `HasRole`
  does exact-match against the JWT roles claim. Hierarchy resolution
  can be added inside `HasRole` without touching callers.
- **Audit logging.** `RequireScope` only logs warnings on 403. A future
  PR will emit structured audit events for every forbidden decision.
- **Webhook signatures.** `payments/webhook` is unauthenticated today.
  A future PR will secure it with either HMAC signatures or a dedicated
  `payments:webhook` scope for M2M.
- **Policy engines (OPA, Casbin).** Not warranted at current scale.
  Pure Go interfaces are enough.

See [the v1 PR](https://github.com/OpenNSW/nsw/pull/561) for the full
phased rollout plan (v2: migrate consignment; v3: remaining handlers;
v4: security hardening).
