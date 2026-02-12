package auth

import "encoding/json"

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
