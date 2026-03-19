package internal

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	DBDriver       string // "sqlite" or "postgres"
	DBPath         string // for sqlite
	DBHost         string // for postgres
	DBPort         string // for postgres
	DBUser         string // for postgres
	DBPassword     string // for postgres
	DBName         string // for postgres
	DBSSLMode      string // for postgres
	FormsPath      string
	DefaultFormID  string
	AllowedOrigins []string
	NSWAPIBaseURL  string
}

func LoadConfig() Config {
	return Config{
		Port:           envOrDefault("OGA_PORT", "8081"),
		DBDriver:       envOrDefault("OGA_DB_DRIVER", "sqlite"),
		DBPath:         envOrDefault("OGA_DB_PATH", "./oga_applications.db"),
		DBHost:         envOrDefault("OGA_DB_HOST", "localhost"),
		DBPort:         envOrDefault("OGA_DB_PORT", "5432"),
		DBUser:         envOrDefault("OGA_DB_USER", "postgres"),
		DBPassword:     envOrDefault("OGA_DB_PASSWORD", "changeme"),
		DBName:         envOrDefault("OGA_DB_NAME", "oga_db"),
		DBSSLMode:      envOrDefault("OGA_DB_SSLMODE", "disable"),
		FormsPath:      envOrDefault("OGA_FORMS_PATH", "./data/forms"),
		DefaultFormID:  envOrDefault("OGA_DEFAULT_FORM_ID", "default"),
		AllowedOrigins: parseOrigins(envOrDefault("OGA_ALLOWED_ORIGINS", "*")),
		NSWAPIBaseURL:  envOrDefault("NSW_API_BASE_URL", "http://localhost:8080/api/v1"),
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
