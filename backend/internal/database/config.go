package database

import (
	"fmt"
	"net/url"
)

// Config holds database connection configuration.
type Config struct {
	Host                   string
	Port                   int
	Username               string
	Password               string
	Name                   string
	SSLMode                string
	MaxIdleConns           int
	MaxOpenConns           int
	MaxConnLifetimeSeconds int
}

func (c Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Username == "" {
		return fmt.Errorf("DB_USERNAME is required")
	}
	if c.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}

// DSN returns the database connection string.
func (c Config) DSN() string {
	// Using the URL format is more robust for handling special characters in passwords.
	// format: postgres://user:password@host:port/dbname?sslmode=disable
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Name,
	}
	query := dsn.Query()
	query.Add("sslmode", c.SSLMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}
