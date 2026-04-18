package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"gorm.io/gorm"
)

// Middleware creates an HTTP middleware that extracts and injects authentication context.
// This middleware:
// 1. Extracts the Authorization header
// 2. Parses the token into a user principal or client principal
// 3. Looks up user context from the database for user principals
// 4. Injects the auth context into the request
//
// If a user has no stored context, AuthContext is still injected
// with User.NSWData initialized to an empty object.
//
// Behavior summary:
// - Missing Authorization header: request proceeds without auth context.
// - Invalid token: request is rejected with 401.
// - Auth dependencies unavailable or DB errors: request is rejected with 500.
//
// This design allows:
// - Public endpoints (no auth required)
// - Protected endpoints (check for context)
// - Optional auth endpoints (use context if available)
func Middleware(authService *AuthService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Debug("no authorization header provided")
				next.ServeHTTP(w, r)
				return
			}

			if tokenExtractor == nil || authService == nil {
				slog.Error("auth middleware dependencies are not initialized")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal_server_error","message":"authentication subsystem not initialized"}`))
				return
			}

			principal, err := tokenExtractor.ExtractPrincipalFromHeader(authHeader)
			if err != nil {
				slog.Warn("failed to extract principal from token", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"invalid authentication token"}`))
				return
			}

			if principal == nil {
				slog.Warn("token extractor returned nil principal")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"invalid authentication token"}`))
				return
			}

			if principal.UserPrincipal == nil && principal.ClientPrincipal == nil {
				slog.Warn("token missing both userPrincipal and clientPrincipal")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"invalid authentication token"}`))
				return
			}

			authCtx := buildAuthContext(principal)
			if principal.UserPrincipal != nil {
				userCtx, err := authService.GetUserContext(principal.UserPrincipal.UserID)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						slog.Debug("no stored user context, creating default context",
							"user_id", principal.UserPrincipal.UserID)
						err = authService.UpsertUserContext(principal.UserPrincipal.UserID, UpsertUserContextPayload{
							Email:    &principal.UserPrincipal.Email,
							OUHandle: &principal.UserPrincipal.OUHandle,
							OUID:     &principal.UserPrincipal.OUID,
							NSWData:  []byte(`{}`),
						})
						if err != nil {
							slog.Error("failed to create default user context", "user_id", principal.UserPrincipal.UserID, "error", err)
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusInternalServerError)
							_, _ = w.Write([]byte(`{"error":"internal_server_error","message":"failed to initialize user context"}`))
							return
						}
					} else {
						slog.Error("failed to get user context from database", "user_id", principal.UserPrincipal.UserID, "error", err)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"error":"internal_server_error","message":"failed to retrieve user context"}`))
						return
					}
					userCtx = &UserContext{
						UserID:   principal.UserPrincipal.UserID,
						Email:    principal.UserPrincipal.Email,
						OUHandle: principal.UserPrincipal.OUHandle,
						OUID:     principal.UserPrincipal.OUID,
						NSWData:  []byte(`{}`),
					}
				}
				authCtx.User = userCtx
			}
			ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func buildAuthContext(principal *Principal) *AuthContext {
	if principal == nil {
		return &AuthContext{}
	}

	switch principal.Type {
	case UserPrincipalType:
		if principal.UserPrincipal == nil {
			return &AuthContext{}
		}
		return &AuthContext{
			User: &UserContext{
				UserID:   principal.UserPrincipal.UserID,
				Email:    principal.UserPrincipal.Email,
				OUHandle: principal.UserPrincipal.OUHandle,
				OUID:     principal.UserPrincipal.OUID,
			},
		}
	case ClientPrincipalType:
		if principal.ClientPrincipal == nil {
			return &AuthContext{}
		}
		return &AuthContext{
			Client: &ClientContext{ClientID: principal.ClientPrincipal.ClientID},
		}
	default:
		if principal.UserPrincipal != nil {
			return &AuthContext{
				User: &UserContext{
					UserID:   principal.UserPrincipal.UserID,
					Email:    principal.UserPrincipal.Email,
					OUHandle: principal.UserPrincipal.OUHandle,
					OUID:     principal.UserPrincipal.OUID,
				},
			}
		}
		if principal.ClientPrincipal != nil {
			return &AuthContext{
				Client: &ClientContext{ClientID: principal.ClientPrincipal.ClientID},
			}
		}
		return &AuthContext{}
	}
}

// RequireAuth returns a middleware that requires authentication.
// If no auth context is found, returns 401 Unauthorized.
// This middleware should be applied to protected endpoints.
//
// Usage:
//
//	mux.Handle("POST /api/protected", auth.RequireAuth(authService, tokenExtractor)(handler))
//
// TODO_JWT_FUTURE: Consider adding:
// - Different auth levels (basic, standard, admin)
// - Claim validation beyond token signature
// - Rate limiting per user
func RequireAuth(authService *AuthService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	authMiddleware := Middleware(authService, tokenExtractor)
	return func(next http.Handler) http.Handler {
		return authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetAuthContext(r.Context()) == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"authentication required"}`))
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}
