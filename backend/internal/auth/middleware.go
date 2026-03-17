package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// Middleware creates an HTTP middleware that extracts and injects authentication context.
// This middleware:
// 1. Extracts the Authorization header
// 2. Parses the token to get the user ID and email
// 3. Looks up the user context from the database
// 4. Injects the auth context into the request
//
// If the user has no stored context (e.g. CHA users), AuthContext is still injected
// with UserContext = nil. Handlers must tolerate a nil UserContext.
//
// If any step fails (missing token, invalid token, user not found),
// the request proceeds without auth context. Handlers should check for context availability.
//
// This design allows:
// - Public endpoints (no auth required)
// - Protected endpoints (check for context)
// - Optional auth endpoints (use context if available)
//
// TODO_JWT_FUTURE: When JWT is implemented:
// - The middleware stays the same
// - Token verification moves to token_parser.go (ExtractClaimsFromHeader)
// - Consider adding additional claims from JWT to AuthContext
// - Consider adding rate limiting based on token/user
// - Consider adding audit logging for auth attempts
func Middleware(authService *AuthService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass auth for external payment callbacks
			if strings.HasPrefix(r.URL.Path, "/api/v1/payments/") && strings.HasSuffix(r.URL.Path, "/callback") {
				next.ServeHTTP(w, r)
				return
			}

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

			claims, err := tokenExtractor.ExtractClaimsFromHeader(authHeader)
			if err != nil {
				slog.Warn("failed to extract claims from token", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"invalid authentication token"}`))
				return
			}

			authCtx := &AuthContext{
				UserID:   claims.UserID,
				Email:    claims.Email,
				OUHandle: claims.OUHandle,
			}

			userCtx, err := authService.GetUserContext(claims.UserID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					slog.Debug("no stored user context, proceeding with nil UserContext",
						"user_id", claims.UserID)
				} else {
					slog.Warn("failed to get user context from database",
						"user_id", claims.UserID, "error", err)
					next.ServeHTTP(w, r)
					return
				}
			} else {
				authCtx.UserContext = userCtx
			}

			ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
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
