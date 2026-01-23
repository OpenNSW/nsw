package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Name     string
	SSLMode  string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	dbPort, err := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	serverPort, err := strconv.Atoi(getEnvOrDefault("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     dbPort,
			Username: getEnvOrDefault("DB_USERNAME", "postgres"),
			Password: os.Getenv("DB_PASSWORD"), // No default for security
			Name:     getEnvOrDefault("DB_NAME", "nsw_db"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			Port: serverPort,
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("DB_USERNAME is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	// URL encode the password to handle special characters
	encodedPassword := url.QueryEscape(c.Password)

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.Username,
		encodedPassword,
		c.Name,
		c.SSLMode,
	)
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
