package auth

import (
	"encoding/json"
	"fmt"
)

// TraderContext represents the context of a trader in the database.
// This model persists trader information and associated metadata.
type TraderContext struct {
	TraderID      string          `gorm:"type:varchar(100);column:trader_id;primaryKey;not null" json:"trader_id"`
	TraderContext json.RawMessage `gorm:"type:jsonb;column:trader_context;serializer:json;not null" json:"trader_context"`
}

// TableName specifies the database table name for TraderContext
func (t *TraderContext) TableName() string {
	return "trader_contexts"
}

// AuthContext represents the authentication context available in a request.
// This is a transient context that is injected into the request by the auth middleware.
// It contains trader information retrieved from the database based on the token.
//
// Future: When JWT is implemented, this struct may be extended to include claims like:
// - TokenIssuedAt (iat)
// - TokenExpiresAt (exp)
// - Additional JWT-specific fields
type AuthContext struct {
	*TraderContext
}

// GetTraderContextMap returns the trader context as a map for convenient access.
// If no context exists, it returns an empty map.
func (ac *AuthContext) GetTraderContextMap() (map[string]any, error) {
	contextMap := make(map[string]any)
	if ac == nil || ac.TraderContext == nil || len(ac.TraderContext.TraderContext) == 0 {
		return contextMap, nil
	}

	if err := json.Unmarshal(ac.TraderContext.TraderContext, &contextMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trader context: %w", err)
	}

	return contextMap, nil
}
