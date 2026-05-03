package auth

import (
	"context"
	"log/slog"
	"net/http"
)

// Middleware creates an HTTP middleware that extracts and injects authentication context.
// This middleware:
// 1. Extracts the Authorization header
// 2. Parses the token into a user principal or client principal
// 3. For user principals on first login, creates a user profile if UserProfileService is provided
// 4. Injects the auth context into the request
//
// Behavior summary:
// - Missing Authorization header: request proceeds without auth context.
// - Invalid token: request is rejected with 401.
// - Auth dependencies unavailable: request is rejected with 500.
// - User principal on first login: automatically creates user profile if service is provided.
//
// This design allows:
// - Public endpoints (no auth required)
// - Protected endpoints (check for context)
// - Optional auth endpoints (use context if available)
// - Generic auth that works with or without a user profile service
func Middleware(userProfileService UserProfileService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Debug("no authorization header provided")
				next.ServeHTTP(w, r)
				return
			}

			if tokenExtractor == nil {
				slog.Error("auth middleware: token extractor not initialized")
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
				// Attempt to create user profile if service is provided (optional on first login).
				// This is a fire-and-forget operation; we proceed even if it fails.
				if userProfileService != nil {
					// Check if the user already exists to avoid creating on every request.
					exists, err := userProfileService.UserExists(principal.UserPrincipal.UserID)
					if err != nil {
						// Error checking existence; log but don't block auth
						slog.Warn("failed to check if user exists; will attempt create anyway", "user_id", principal.UserPrincipal.UserID, "error", err)
						create_err := userProfileService.CreateUser(
							principal.UserPrincipal.UserID,
							principal.UserPrincipal.Email,
							func() string {
								if principal.UserPrincipal.PhoneNumber != nil {
									return *principal.UserPrincipal.PhoneNumber
								}
								return ""
							}(),
							principal.UserPrincipal.OUID,
						)
						if create_err != nil {
							slog.Error("failed to create user profile on first login", "user_id", principal.UserPrincipal.UserID, "error", create_err)
						}
					} else if exists {
						// user exists - no need to create
						slog.Debug("user already exists - skipping creation", "user_id", principal.UserPrincipal.UserID)
					} else {
						// user does not exist - attempt to create
						create_err := userProfileService.CreateUser(
							principal.UserPrincipal.UserID,
							principal.UserPrincipal.Email,
							func() string {
								if principal.UserPrincipal.PhoneNumber != nil {
									return *principal.UserPrincipal.PhoneNumber
								}
								return ""
							}(),
							principal.UserPrincipal.OUID,
						)
						if create_err != nil {
							slog.Error("failed to create user profile on first login", "user_id", principal.UserPrincipal.UserID, "error", create_err)
						} else {
							slog.Debug("created user profile on first login", "user_id", principal.UserPrincipal.UserID)
						}
					}
				} else {
					slog.Debug("user profile service not provided - skipping user creation on first login")
				}

				// Build UserContext from the principal (no preloading of NSWData).
				authCtx.User = &UserContext{
					UserID: principal.UserPrincipal.UserID,
					Email:  principal.UserPrincipal.Email,
					PhoneNumber: func() string {
						if principal.UserPrincipal.PhoneNumber != nil {
							return *principal.UserPrincipal.PhoneNumber
						}
						return ""
					}(),
					OUID:  principal.UserPrincipal.OUID,
					Roles: principal.UserPrincipal.Roles,
				}
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
		phoneNumber := ""
		if principal.UserPrincipal.PhoneNumber != nil {
			phoneNumber = *principal.UserPrincipal.PhoneNumber
		}
		return &AuthContext{
			User: &UserContext{
				UserID:      principal.UserPrincipal.UserID,
				Email:       principal.UserPrincipal.Email,
				PhoneNumber: phoneNumber,
				OUID:        principal.UserPrincipal.OUID,
				Roles:       principal.UserPrincipal.Roles,
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
			phoneNumber := ""
			if principal.UserPrincipal.PhoneNumber != nil {
				phoneNumber = *principal.UserPrincipal.PhoneNumber
			}
			return &AuthContext{
				User: &UserContext{
					UserID:      principal.UserPrincipal.UserID,
					Email:       principal.UserPrincipal.Email,
					PhoneNumber: phoneNumber,
					OUID:        principal.UserPrincipal.OUID,
					Roles:       principal.UserPrincipal.Roles,
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
//	mux.Handle("POST /api/protected", auth.RequireAuth(userProfileService, tokenExtractor)(handler))
//
// TODO_JWT_FUTURE: Consider adding:
// - Different auth levels (basic, standard, admin)
// - Claim validation beyond token signature
// - Rate limiting per user
func RequireAuth(userProfileService UserProfileService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	authMiddleware := Middleware(userProfileService, tokenExtractor)
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
