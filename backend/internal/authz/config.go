package authz

import "fmt"

// Config configures the authorization manager. Both maps are static
// configuration typically loaded once at application startup. An empty
// configuration is valid and produces a manager where every HasScope check
// returns false; this is useful in environments where authorization is
// still being rolled out per feature.
type Config struct {
	// RoleScopes maps a JWT role claim value to the scopes that role grants.
	// Applied to principals authenticated via OAuth2 authorization_code grant.
	// Example: {"trader": ["consignments:read", "consignments:write"]}.
	RoleScopes map[string][]string

	// ClientScopes maps an OAuth2 client_id to the scopes that client is
	// allowed to use. Applied to principals authenticated via OAuth2
	// client_credentials grant.
	// Example: {"lankapay-m2m": ["payments:webhook"]}.
	ClientScopes map[string][]string
}

// Validate returns an error if the configuration contains structural
// problems. Empty maps are allowed; empty scope strings are not.
func (c Config) Validate() error {
	for role, scopes := range c.RoleScopes {
		if role == "" {
			return fmt.Errorf("AUTHZ_ROLE_SCOPES: role name must not be empty")
		}
		for _, scope := range scopes {
			if scope == "" {
				return fmt.Errorf("AUTHZ_ROLE_SCOPES: role %q has an empty scope", role)
			}
		}
	}
	for clientID, scopes := range c.ClientScopes {
		if clientID == "" {
			return fmt.Errorf("AUTHZ_CLIENT_SCOPES: client id must not be empty")
		}
		for _, scope := range scopes {
			if scope == "" {
				return fmt.Errorf("AUTHZ_CLIENT_SCOPES: client %q has an empty scope", clientID)
			}
		}
	}
	return nil
}
