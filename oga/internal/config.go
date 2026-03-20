package internal

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	DBPath         string
	FormsPath      string
	DefaultFormID  string
	AllowedOrigins []string
	NSWAPIBaseURL  string
}

func LoadConfig() Config {
	return Config{
		Port:      envOrDefault("OGA_PORT", "8081"),
		DBPath:    envOrDefault("OGA_DB_PATH", "./oga_applications.db"),
		FormsPath: envOrDefault("OGA_FORMS_PATH", "./data/forms"),
		// If Meta is not set in the request from NSW, the default form will be used
		DefaultFormID: envOrDefault("OGA_DEFAULT_FORM_ID", "default"),
		// TODO: when productionization, need to remove the '*' (Allowing All Origins)
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
