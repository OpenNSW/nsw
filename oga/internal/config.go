package internal

import (
	"log"
	"os"
	"strings"

	"github.com/OpenNSW/nsw/oga/internal/database"
)

// Config holds the application configuration
type Config struct {
	Port           string
	DB             database.Config
	FormsPath      string
	DefaultFormID  string
	AllowedOrigins []string
	NSWAPIBaseURL  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	driver := envOrDefault("OGA_DB_DRIVER", "sqlite")

	// Fetch the password directly using the fallback helper
	password := firstEnv("OGA_DB_PASSWORD", "DB_PASSWORD", "")

	// The Fail-Fast Security Check for Production (Postgres)
	if driver == "postgres" && password == "" {
		log.Fatal("FATAL: Database password secret is missing! OGA_DB_PASSWORD or DB_PASSWORD is required for postgres.")
	}

	// Fallback exclusively for local SQLite development
	if password == "" {
		password = "changeme"
	}

	return Config{
		Port: envOrDefault("OGA_PORT", "8081"),
		DB: database.Config{
			Driver:   driver,
			Path:     envOrDefault("OGA_DB_PATH", "./oga_applications.db"),
			Host:     firstEnv("OGA_DB_HOST", "DB_HOST", "localhost"),
			Port:     firstEnv("OGA_DB_PORT", "DB_PORT", "5432"),
			User:     firstEnv("OGA_DB_USER", "DB_USERNAME", "postgres"),
			Password: password, // Uses the validated password
			Name:     firstEnv("OGA_DB_NAME", "DB_NAME", "oga_db"),
			SSLMode:  envOrDefault("OGA_DB_SSLMODE", "disable"),
		},
		FormsPath:      envOrDefault("OGA_FORMS_PATH", "./data/forms"),
		DefaultFormID:  envOrDefault("OGA_DEFAULT_FORM_ID", "default"),
		AllowedOrigins: parseOrigins(envOrDefault("OGA_ALLOWED_ORIGINS", "*")),
		NSWAPIBaseURL:  envOrDefault("NSW_API_BASE_URL", "http://localhost:8080/api/v1"),
	}
}

// firstEnv checks multiple environment variables in order, returning the first one found, or the fallback.
func firstEnv(key1, key2, fallback string) string {
	if v := os.Getenv(key1); v != "" {
		return v
	}
	if v := os.Getenv(key2); v != "" {
		return v
	}
	return fallback
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}
	parts := strings.Split(origins, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}
