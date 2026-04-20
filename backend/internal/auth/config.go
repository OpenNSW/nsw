package auth

import "fmt"

type Config struct {
	JWKSURL               string
	Issuer                string
	Audience              string
	ClientID              string
	InsecureSkipTLSVerify bool
}

func (c *Config) Validate() error {
	if c.JWKSURL == "" {
		return fmt.Errorf("AUTH_JWKS_URL is required")
	}
	if c.Issuer == "" {
		return fmt.Errorf("AUTH_ISSUER is required")
	}
	if c.Audience == "" {
		return fmt.Errorf("AUTH_AUDIENCE is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("AUTH_CLIENT_ID is required")
	}
	return nil
}
