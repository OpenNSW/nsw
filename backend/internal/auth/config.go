package auth

import (
	"fmt"

	"github.com/OpenNSW/nsw/internal/validation"
)

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
	if err := validation.HTTPURL("AUTH_JWKS_URL", c.JWKSURL); err != nil {
		return err
	}
	if c.Issuer == "" {
		return fmt.Errorf("AUTH_ISSUER is required")
	}
	if err := validation.HTTPURL("AUTH_ISSUER", c.Issuer); err != nil {
		return err
	}
	if c.Audience == "" {
		return fmt.Errorf("AUTH_AUDIENCE is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("AUTH_CLIENT_ID is required")
	}
	return nil
}
