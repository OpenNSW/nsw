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
// by the auth middleware.
// For user principals, UserID and identity fields are set.
// For client principals (M2M), ClientID is set and UserID may be nil.
// UserContext is nullable — users without a DB entry are allowed.
type AuthContext struct {
	UserID      *string      `json:"userId,omitempty"`
	Email       *string      `json:"email,omitempty"`
	OUHandle    *string      `json:"ouHandle,omitempty"`
	OUID        *string      `json:"ouId,omitempty"`
	ClientID    *string      `json:"clientId,omitempty"`
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
// Returns nil if no auth context is available (for example: public route,
// missing auth header, or middleware not applied).
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
