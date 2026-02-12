package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"gorm.io/gorm"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// AuthContextKey is the key for storing AuthContext in request context
	AuthContextKey ContextKey = "authContext"
)

// Middleware creates an HTTP middleware that extracts and injects authentication context.
// This middleware:
// 1. Extracts the Authorization header
// 2. Parses the token to get the trader ID
// 3. Looks up the trader context from the database
// 4. Injects the trader context into the request
//
// If any step fails (missing token, invalid token, trader not found),
// the request proceeds without auth context. Handlers should check for context availability.
//
// This design allows:
// - Public endpoints (no auth required)
// - Protected endpoints (check for context)
// - Optional auth endpoints (use context if available)
//
// TODO_JWT_FUTURE: When JWT is implemented:
// - The middleware stays the same
// - Token verification moves to token_parser.go (ExtractTraderIDFromHeader)
// - Consider adding additional claims from JWT to AuthContext
// - Consider adding rate limiting based on token/trader
// - Consider adding audit logging for auth attempts
func Middleware(authService *AuthService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")

			// If no Authorization header, continue without auth context
			if authHeader == "" {
				slog.Debug("no authorization header provided")
				next.ServeHTTP(w, r)
				return
			}

			// Extract trader ID from token
			traderID, err := tokenExtractor.ExtractTraderIDFromHeader(authHeader)
			if err != nil {
				slog.Warn("failed to extract trader ID from token",
					"error", err,
					"auth_header_length", len(authHeader),
				)
				next.ServeHTTP(w, r)
				return
			}

			// Get trader context from database
			traderCtx, err := authService.GetTraderContext(traderID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// Trader doesn't have context yet - create empty context
					slog.Info("trader context not found, initializing empty context",
						"trader_id", traderID,
					)
					traderCtx = &TraderContext{
						TraderID:      traderID,
						TraderContext: json.RawMessage(`{}`),
					}
				} else {
					// Database error - log and continue without auth context
					slog.Warn("failed to get trader context from database",
						"trader_id", traderID,
						"error", err,
					)
					next.ServeHTTP(w, r)
					return
				}
			}

			// Wrap the trader context in AuthContext
			authCtx := &AuthContext{
				TraderContext: traderCtx,
			}

			// Inject auth context into request context
			ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
			r = r.WithContext(ctx)

			slog.Debug("auth context injected successfully",
				"trader_id", traderID,
			)

			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthContext extracts the AuthContext from a request context.
// Returns nil if no auth context is available (request had no valid token).
//
// Usage in handlers:
//
//	authCtx := auth.GetAuthContext(r.Context())
//	if authCtx == nil {
//	    // Handle unauthorized request
//	}
//	traderID := authCtx.TraderID
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
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
// - Rate limiting per trader
func RequireAuth(authService *AuthService, tokenExtractor *TokenExtractor) func(http.Handler) http.Handler {
	// Create the auth middleware once, not on every request
	authMiddleware := Middleware(authService, tokenExtractor)

	return func(next http.Handler) http.Handler {
		return authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if auth context is available
			authCtx := GetAuthContext(r.Context())
			if authCtx == nil {
				slog.Warn("authentication required but not provided",
					"method", r.Method,
					"path", r.URL.Path,
				)
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"authentication required"}`))
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}
