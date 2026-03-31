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

type ContextKey string

const AuthContextKey ContextKey = "auth_context"

const (
	RoleQueryTrader = "trader"
	RoleQueryCHA    = "cha"
)

type AuthContext struct {
	UserID      *string      `json:"userId"`
	Email       string       `json:"email"`
	OUHandle    string       `json:"ouHandle"`
	UserContext *UserContext `json:"userContext,omitempty"`
	ClientID    string       `json:"clientId"`
	Groups      []string     `json:"groups"` // e.g. ["Trader", "CHA"]
	IsM2M       bool         `json:"isM2M"`  // True if Client Credentials grant
}

func (c *AuthContext) HasGroup(group string) bool {
	if c == nil {
		return false
	}
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

func (c *AuthContext) GetUserContextMap() (map[string]any, error) {
	m := make(map[string]any)
	if c == nil || c.UserContext == nil || len(c.UserContext.UserContext) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(c.UserContext.UserContext, &m); err != nil {
		return nil, err
	}
	return m, nil
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
//	userID := authCtx.UserID
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}

func FromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx, ok
}
