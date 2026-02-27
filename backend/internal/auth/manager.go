package auth

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/config"
)

// Manager handles all authentication-related operations and middleware setup.
// It encapsulates the auth service, token extraction, and middleware creation,
// providing a clean interface for the HTTP server to use.
//
// This manager pattern keeps auth logic self-contained and makes main.go cleaner.
//
// TODO_JWT_FUTURE: When JWT is implemented, this manager will be extended to:
// - Load JWT configuration (secret/public key)
// - Initialize JWT validators
// - Manage token caching
// - Handle key rotation
type Manager struct {
	service        *AuthService
	tokenExtractor *TokenExtractor
	middleware     func(http.Handler) http.Handler
}

// NewManager creates and initializes a new auth manager.
// This is the single entry point for all auth initialization in the application.
//
// Usage in main.go:
//
//	authManager := auth.NewManager(db)
//	handler := middleware.CORS(&cfg.CORS)(authManager.Middleware()(mux))
//
// This centralizes all auth setup and makes it easy to extend with JWT configuration later.
func NewManager(db *gorm.DB, authConfig config.AuthConfig) (*Manager, error) {
	slog.Info("initializing auth manager")

	// Initialize auth service
	service := NewAuthService(db)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	if authConfig.InsecureSkipTLSVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Initialize token extractor
	tokenExtractor, err := NewTokenExtractorWithClient(
		authConfig.JWKSURL,
		authConfig.Issuer,
		authConfig.Audience,
		authConfig.ClientID,
		httpClient,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token extractor: %w", err)
	}

	// Create middleware
	middleware := Middleware(service, tokenExtractor)

	m := &Manager{
		service:        service,
		tokenExtractor: tokenExtractor,
		middleware:     middleware,
	}

	slog.Debug("auth manager initialized successfully")
	return m, nil
}

// Middleware returns the auth middleware function.
// This middleware extracts and injects authentication context into requests.
//
// Usage:
//
//	handler := authManager.Middleware()(mux)
//
// The middleware:
// 1. Extracts Authorization header
// 2. Parses token to get trader ID
// 3. Looks up trader context from database
// 4. Injects context into request
//
// Gracefully degrades if auth is missing or fails.
func (m *Manager) Middleware() func(http.Handler) http.Handler {
	return m.middleware
}

// Service returns the auth service for direct use if needed.
// Most applications won't need this - use the middleware instead.
// Useful for:
// - Direct trader context lookups
// - Updating trader information
// - Admin operations
//
// Example:
//
//	authService := authManager.Service()
//	context, err := authService.GetTraderContext("TRADER-001")
func (m *Manager) Service() *AuthService {
	return m.service
}

// RequireAuthMiddleware returns a middleware that requires authentication.
// If no auth context is found, returns 401 Unauthorized.
// Use this for protected endpoints.
//
// Usage:
//
//	mux.Handle("POST /api/protected",
//	    authManager.RequireAuthMiddleware()(handler),
//	)
//
// TODO_JWT_FUTURE: Could extend this to support role-based access:
// authManager.RequireAuthMiddleware("admin", "exporter")
func (m *Manager) RequireAuthMiddleware() func(http.Handler) http.Handler {
	return RequireAuth(m.service, m.tokenExtractor)
}

// OptionalAuthMiddleware returns a middleware for endpoints that work with or without auth.
// This is the same as the basic Middleware() - included for semantic clarity.
//
// Usage:
//
//	mux.Handle("GET /api/products",
//	    authManager.OptionalAuthMiddleware()(handler),
//	)
//
// The handler can check if auth context is available and personalize response.
func (m *Manager) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return m.middleware
}

// GetTraderContext is a convenience method to look up trader context directly.
// Useful for non-request operations (e.g., background jobs, admin commands).
//
// Example:
//
//	authManager := auth.NewManager(db)
//	ctx, err := authManager.GetTraderContext("TRADER-001")
//
// For request-based operations, use auth.GetAuthContext(r.Context()) in handlers instead.
func (m *Manager) GetTraderContext(traderID string) (*TraderContext, error) {
	return m.service.GetTraderContext(traderID)
}

// UpdateTraderContext is a convenience method to update trader context directly.
// Useful for admin operations or background jobs.
//
// Example:
//
//	authManager := auth.NewManager(db)
//	newContext := json.RawMessage(`{"status": "verified"}`)
//	err := authManager.UpdateTraderContext("TRADER-001", newContext)
//
// For request-based operations, use a handler with auth context instead.
func (m *Manager) UpdateTraderContext(traderID string, context interface{}) error {
	// Handle both json.RawMessage and other types
	var data []byte
	var err error
	switch v := context.(type) {
	case []byte:
		data = v
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal context: %w", err)
		}
	}
	return m.service.UpdateTraderContext(traderID, data)
}

// Health checks if the auth system is functioning properly.
// Performs a sample database query to verify:
// 1. Database connection is alive
// 2. trader_contexts table exists
// 3. Auth service can perform lookups
//
// Usage in server startup:
//
//	authManager := auth.NewManager(db)
//	if err := authManager.Health(); err != nil {
//	    log.Fatalf("auth system health check failed: %v", err)
//	}
//
// Returns an error if anything is wrong, allowing graceful failure at startup.
func (m *Manager) Health() error {
	// Try to query trader_contexts table (even if empty)
	var count int64
	result := m.service.db.Model(&TraderContext{}).Count(&count)
	if result.Error != nil {
		slog.Error("auth health check failed",
			"error", result.Error,
		)
		return result.Error
	}

	slog.Info("auth health check passed", "traders_count", count)
	return nil
}

// Close performs any cleanup needed for the auth manager.
// Currently a no-op, but provided for future extensibility
// (e.g., closing JWT validators, flushing caches, logging stats).
//
// Usage:
//
//	authManager := auth.NewManager(db)
//	defer authManager.Close()
func (m *Manager) Close() error {
	slog.Debug("auth manager closing")
	// TODO_JWT_FUTURE: Add cleanup logic for JWT validators, caches, etc.
	return nil
}
