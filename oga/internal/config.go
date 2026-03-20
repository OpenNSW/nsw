package internal

import (
	"os"
	"strings"

	"github.com/OpenNSW/nsw/oga/internal/database"
)

type Config struct {
	Port           string
	DB             database.Config
	FormsPath      string
	DefaultFormID  string
	AllowedOrigins []string
}

func LoadConfig() Config {
	return Config{
		Port: envOrDefault("OGA_PORT", "8081"),
		DB: database.Config{
			Driver:   envOrDefault("OGA_DB_DRIVER", "sqlite"),
			Path:     envOrDefault("OGA_DB_PATH", "./oga_applications.db"),
			Host:     envOrDefault("OGA_DB_HOST", "localhost"),
			Port:     envOrDefault("OGA_DB_PORT", "5432"),
			User:     envOrDefault("OGA_DB_USER", "postgres"),
			Password: envOrDefault("OGA_DB_PASSWORD", "changeme"),
			Name:     envOrDefault("OGA_DB_NAME", "oga_db"),
			SSLMode:  envOrDefault("OGA_DB_SSLMODE", "disable"),
		},
		FormsPath:      envOrDefault("OGA_FORMS_PATH", "./data/forms"),
		DefaultFormID:  envOrDefault("OGA_DEFAULT_FORM_ID", "default"),
		AllowedOrigins: parseOrigins(envOrDefault("OGA_ALLOWED_ORIGINS", "*")),
	}
}

// parseOrigins splits a comma-separated list of origins.
func parseOrigins(s string) []string {
	var origins []string
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
