package auth

import (
	"context"
)

// UserProfileService defines the contract for managing user profiles.
// Implementations are responsible for persisting and managing user records in their system.
//
// This interface is OPTIONAL when using the auth package. If not provided (nil),
// user creation on first login will be skipped. This allows:
//
// 1. Systems that don't track user profiles - just use auth for token validation
// 2. Systems that manage user profiles separately - implement this interface
// 3. Systems that handle user creation elsewhere - pass nil
//
// Example implementation:
//
//	type MyUserService struct {
//	    db *sql.DB
//	}
//
//	func (s *MyUserService) CreateUser(userID, email, phone, organizationID string) error {
//	    // Your implementation to persist user
//	    return s.db.Exec("INSERT INTO users ...", userID, email, phone, organizationID).Error
//	}
//
//	authManager := auth.NewManager(myUserService, cfg.Auth)  // myUserService can be nil
type UserProfileService interface {
	// CreateUser creates or updates a user profile.
	// Parameters:
	//   - userID: unique user identifier (required)
	//   - email: user's email address (required)
	//   - phone: user's phone number (can be empty)
	//   - organizationID: organization/tenant identifier (required)
	//
	// Implementation notes:
	//   - Should be idempotent: calling multiple times with same userID should be safe
	//   - Called during first login after token validation
	//   - Errors are logged but don't block authentication
	//   - Should not return error if user already exists
	CreateUser(userID, email, phone, organizationID string) error

	// UserExists checks if a user profile exists for the given userID.
	// Returns true if the user exists, false if not, or an error on failure.
	UserExists(userID string) (bool, error)
}

// UserContext represents a user principal's runtime context injected into each request.
// It includes identity fields and principal-derived roles.
// Note: Per-request NSWData is not persisted here; services requiring user metadata
// should call the user profile service on-demand.
type UserContext struct {
	UserID      string   `json:"userId"`
	Email       string   `json:"email"`
	PhoneNumber string   `json:"phoneNumber"`
	OUID        string   `json:"ouId"`
	Roles       []string `json:"roles"`
}

// ClientContext represents a machine client's context.
type ClientContext struct {
	ClientID string
}

// AuthContext is the transient authentication context injected into each request
// by the auth middleware.
// For user principals, User contains identity fields and roles.
// For client principals (M2M), Client is set.
type AuthContext struct {
	User   *UserContext
	Client *ClientContext
}

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const AuthContextKey ContextKey = "authContext"

// GetAuthContext extracts the AuthContext from a request context.
// Returns nil if no auth context is available (for example: public route,
// missing auth header, or middleware not applied).
//
// Usage in handlers:
//
//	authCtx := auth.GetAuthContext(r.Context())
//	if authCtx == nil {
//	    // Handle unauthorized request
//	}
//	userID := authCtx.User.UserID
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}
