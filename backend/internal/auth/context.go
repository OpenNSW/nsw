package auth

import (
	"context"
	"encoding/json"
)

// UserContext represents a user's stored context in the database.
type UserContext struct {
	UserID      string          `gorm:"type:varchar(100);column:user_id;primaryKey;not null" json:"userId"`
	UserContext json.RawMessage `gorm:"type:jsonb;column:user_context;serializer:json;not null" json:"userContext"`
}

func (t *UserContext) TableName() string {
	return "user_contexts"
}

// AuthContext is the transient authentication context injected into each request
// by the auth middleware. UserID is always set (from the JWT sub claim).
// UserContext is nullable — CHAs and other non-trader roles may not have a DB entry.
type AuthContext struct {
	UserID      string       `json:"userId"`
	Email       string       `json:"email"`
	OUHandle    string       `json:"ouHandle"`
	UserContext *UserContext `json:"userContext,omitempty"`
}

// GetUserContextMap returns the stored user context as a map.
// Returns an empty map when no context is available.
func (ac *AuthContext) GetUserContextMap() (map[string]any, error) {
	m := make(map[string]any)
	if ac == nil || ac.UserContext == nil || len(ac.UserContext.UserContext) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(ac.UserContext.UserContext, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const AuthContextKey ContextKey = "authContext"

// GetAuthContext extracts the AuthContext from a request context.
// Returns nil if no auth context is available (request had no valid token).
//
// Usage in handlers:
//
//	authCtx := auth.GetAuthContext(r.Context())
//	if authCtx == nil {
//	    // Handle unauthorized request
//	}
//	userID := authCtx.UserID
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}
