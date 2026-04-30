package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/temporal"
	"github.com/OpenNSW/nsw/internal/uploads"
	"github.com/OpenNSW/nsw/internal/validation"
)

// Config holds all configuration for the application
type Config struct {
	Database     database.Config
	Server       ServerConfig
	CORS         CORSConfig
	Storage      uploads.Config
	Auth         auth.Config
	Notification NotificationConfig
	Temporal     temporal.Config
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port               int
	ServiceURL         string
	ServicesConfigPath string
	Debug              bool
	LogLevel           slog.Level
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type NotificationConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPSender   string
	TemplateRoot string
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
		Database: database.Config{
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
			Port:               serverPort,
			ServiceURL:         getEnvOrDefault("SERVICE_URL", fmt.Sprintf("http://localhost:%d", serverPort)),
			ServicesConfigPath: getEnvOrDefault("SERVICES_CONFIG_PATH", "configs/services.json"),
			Debug:              getBoolOrDefault("SERVER_DEBUG", true),
			LogLevel:           parseLogLevel(getEnvOrDefault("SERVER_LOG_LEVEL", "info")),
		},
		CORS: CORSConfig{
			AllowedOrigins:   parseCommaSeparated(getEnvOrDefault("CORS_ALLOWED_ORIGINS", "*")),
			AllowedMethods:   parseCommaSeparated(getEnvOrDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")),
			AllowedHeaders:   parseCommaSeparated(getEnvOrDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")),
			AllowCredentials: getBoolOrDefault("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getIntOrDefault("CORS_MAX_AGE", 3600),
		},
		Storage: uploads.Config{
			Type:           strings.TrimSpace(getEnvOrDefault("STORAGE_TYPE", "local")),
			LocalBaseDir:   getEnvOrDefault("STORAGE_LOCAL_BASE_DIR", "./bucket"),
			LocalPublicURL: getEnvOrDefault("STORAGE_LOCAL_PUBLIC_URL", getEnvOrDefault("SERVICE_URL", fmt.Sprintf("http://localhost:%d", serverPort))),
			S3Endpoint:     os.Getenv("STORAGE_S3_ENDPOINT"),
			S3Bucket:       getEnvOrDefault("STORAGE_S3_BUCKET", "nsw-uploads"),
			S3Region:       getEnvOrDefault("STORAGE_S3_REGION", "us-east-1"),
			S3AccessKey:    os.Getenv("STORAGE_S3_ACCESS_KEY"),
			S3SecretKey:    os.Getenv("STORAGE_S3_SECRET_KEY"),
			S3UseSSL:       getBoolOrDefault("STORAGE_S3_USE_SSL", true),
			S3PublicURL:    os.Getenv("STORAGE_S3_PUBLIC_URL"),
			LocalPutSecret: getEnvOrDefault("STORAGE_LOCAL_PUT_SECRET", "local-dev-secret"),
			PresignTTL:     parseDurationOrDefault(getEnvOrDefault("STORAGE_PRESIGN_TTL", "15m"), 15*time.Minute),
		},
		Auth: auth.Config{
			JWKSURL:               getEnvOrDefault("AUTH_JWKS_URL", "https://localhost:8090/oauth2/jwks"),
			Issuer:                getEnvOrDefault("AUTH_ISSUER", "https://localhost:8090"),
			Audience:              getEnvOrDefault("AUTH_AUDIENCE", "NSW_API"),
			ClientIDs:             parseCommaSeparated(getEnvOrDefault("AUTH_CLIENT_IDS", "TRADER_PORTAL_APP,FCAU_TO_NSW,NPQS_TO_NSW,IRD_TO_NSW")),
			InsecureSkipTLSVerify: getBoolOrDefault("AUTH_JWKS_INSECURE_SKIP_VERIFY", false),
		},
		Notification: NotificationConfig{
			SMTPHost:     getEnvOrDefault("EMAIL_SMTP_HOST", "localhost"),
			SMTPPort:     getIntOrDefault("EMAIL_SMTP_PORT", 587),
			SMTPUsername: os.Getenv("EMAIL_SMTP_USERNAME"),
			SMTPPassword: os.Getenv("EMAIL_SMTP_PASSWORD"),
			SMTPSender:   getEnvOrDefault("EMAIL_SMTP_SENDER", "noreply@nsw.local"),
			TemplateRoot: getEnvOrDefault("EMAIL_TEMPLATE_ROOT", "./configs/email-templates"),
		},
		Temporal: temporal.Config{
			Host:      getEnvOrDefault("TEMPORAL_HOST", "localhost"),
			PortRaw:   getEnvOrDefault("TEMPORAL_PORT", "7233"),
			Namespace: getEnvOrDefault("TEMPORAL_NAMESPACE", "default"),
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
	if strings.TrimSpace(c.Server.ServiceURL) == "" {
		return fmt.Errorf("SERVICE_URL is required")
	}
	if err := validation.HTTPURL("SERVICE_URL", c.Server.ServiceURL); err != nil {
		return err
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("invalid storage configuration: %w", err)
	}
	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("invalid auth configuration: %w", err)
	}
	if err := c.Temporal.Validate(); err != nil {
		return fmt.Errorf("invalid temporal configuration: %w", err)
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS is required")
	}
	for _, origin := range c.CORS.AllowedOrigins {
		if origin == "*" {
			if !c.Server.Debug {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain '*' in production (SERVER_DEBUG=false)")
			}
			continue
		}
		if err := validation.HTTPURL("CORS_ALLOWED_ORIGINS", origin); err != nil {
			return err
		}
	}
	return nil
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

// parseDurationOrDefault returns the time.Duration value of a string or a default value
func parseDurationOrDefault(value string, defaultValue time.Duration) time.Duration {
	if value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
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

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
