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

type AuthContext struct {
	UserID      *string      // nil if M2M
	Email       string       // optional/empty for M2M
	OUHandle    string       // optional/empty for M2M
	ClientID    string       // Always present
	Groups      []string     // e.g., ["Trader", "CHA"]
	IsM2M       bool         // True if Client Credentials grant
	UserContext *UserContext // Extended internal nsw database record
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
