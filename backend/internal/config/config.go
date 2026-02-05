package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	CORS     CORSConfig
	Storage  StorageConfig
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
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

// ServerConfig holds server configuration
type ServerConfig struct {
	Port       int
	ServiceURL string
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type StorageConfig struct {
	Type           string // "local" or "s3"
	LocalBaseDir   string
	LocalPublicURL string
	S3Endpoint     string
	S3Bucket       string
	S3Region       string
	S3AccessKey    string
	S3SecretKey    string
	S3UseSSL       bool
	S3PublicURL    string
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
			Host:                   getEnvOrDefault("DB_HOST", "localhost"),
			Port:                   dbPort,
			Username:               getEnvOrDefault("DB_USERNAME", "postgres"),
			Password:               os.Getenv("DB_PASSWORD"), // No default for security
			Name:                   getEnvOrDefault("DB_NAME", "nsw_db"),
			SSLMode:                getEnvOrDefault("DB_SSLMODE", "disable"),
			MaxIdleConns:           getIntOrDefault("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:           getIntOrDefault("DB_MAX_OPEN_CONNS", 100),
			MaxConnLifetimeSeconds: getIntOrDefault("DB_MAX_CONN_LIFETIME_SECONDS", 3600),
		},
		Server: ServerConfig{
			Port:       serverPort,
			ServiceURL: getEnvOrDefault("SERVICE_URL", fmt.Sprintf("http://localhost:%d", serverPort)),
		},
		CORS: CORSConfig{
			AllowedOrigins:   parseCommaSeparated(getEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
			AllowedMethods:   parseCommaSeparated(getEnvOrDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")),
			AllowedHeaders:   parseCommaSeparated(getEnvOrDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")),
			AllowCredentials: getBoolOrDefault("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getIntOrDefault("CORS_MAX_AGE", 3600),
		},
		Storage: StorageConfig{
			Type:           getEnvOrDefault("STORAGE_TYPE", "local"),
			LocalBaseDir:   getEnvOrDefault("STORAGE_LOCAL_BASE_DIR", "./uploads"),
			LocalPublicURL: getEnvOrDefault("STORAGE_LOCAL_PUBLIC_URL", "/uploads"),
			S3Endpoint:     os.Getenv("STORAGE_S3_ENDPOINT"),
			S3Bucket:       getEnvOrDefault("STORAGE_S3_BUCKET", "nsw-uploads"),
			S3Region:       getEnvOrDefault("STORAGE_S3_REGION", "us-east-1"),
			S3AccessKey:    os.Getenv("STORAGE_S3_ACCESS_KEY"),
			S3SecretKey:    os.Getenv("STORAGE_S3_SECRET_KEY"),
			S3UseSSL:       getBoolOrDefault("STORAGE_S3_USE_SSL", true),
			S3PublicURL:    os.Getenv("STORAGE_S3_PUBLIC_URL"),
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

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntOrDefault returns the integer value of an environment variable or a default value
func getIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getBoolOrDefault returns the boolean value of an environment variable or a default value
func getBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// parseCommaSeparated splits a comma-separated string into a slice of trimmed strings
func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
