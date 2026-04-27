package auth

import (
	"context"
	"encoding/json"
)

// UserRecord represents a user's stored context in the database.
// This model intentionally excludes roles; roles are derived from the principal.
type UserRecord struct {
	UserID      string          `gorm:"type:varchar(100);column:user_id;primaryKey;not null" json:"userId"`
	Email       string          `gorm:"type:varchar(255);column:email" json:"email"`
	PhoneNumber string          `gorm:"type:varchar(20);column:phone_number" json:"phoneNumber"`
	OUID        string          `gorm:"type:varchar(255);column:ou_id" json:"ouId"`
	NSWData     json.RawMessage `gorm:"type:jsonb;column:nsw_data" json:"nswData"`
}

func (t *UserRecord) TableName() string {
	return "user_records"
}

// UserContext represents a user principal's runtime context.
// It includes principal-derived roles in addition to persisted user context fields.
type UserContext struct {
	UserID      string          `json:"userId"`
	Email       string          `json:"email"`
	PhoneNumber string          `json:"phoneNumber"`
	OUID        string          `json:"ouId"`
	Roles       []string        `json:"roles"`
	NSWData     json.RawMessage `json:"nswData"`
}

func (t *UserContext) TableName() string {
	return "user_contexts"
}

// ClientContext represents a machine client's context.
type ClientContext struct {
	ClientID string
}

// AuthContext is the transient authentication context injected into each request
// by the auth middleware.
// For user principals, UserID and identity fields are set.
// For client principals (M2M), ClientID is set.
type AuthContext struct {
	User   *UserContext
	Client *ClientContext
}

// GetUserContextMap returns the stored user context as a map.
// Returns an empty map when no context is available.
func (ac *AuthContext) GetUserContextMap() (map[string]any, error) {
	m := make(map[string]any)
	if ac == nil || ac.User == nil {
		return m, nil
	}
	if err := json.Unmarshal(ac.User.NSWData, &m); err != nil {
		return nil, err
	}
	return m, nil
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
