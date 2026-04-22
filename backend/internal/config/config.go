package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Database     DatabaseConfig
	Server       ServerConfig
	CORS         CORSConfig
	Storage      StorageConfig
	Auth         AuthConfig
	Notification NotificationConfig
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
	LocalPutSecret string
	PresignTTL     time.Duration
}

type AuthConfig struct {
	JWKSURL               string
	Issuer                string
	Audience              string
	ClientIDs             []string
	InsecureSkipTLSVerify bool
}

type NotificationConfig struct {
	// External email HTTP service
	EmailServiceURL   string
	EmailServiceToken string

	// GovSMS
	GovSMSBaseURL  string
	GovSMSUsername string
	GovSMSPassword string
	GovSMSSIDCode  string

	// Template roots
	EmailTemplateRoot string
	SMSTemplateRoot   string
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

	authJWKSURL := getEnvOrDefault("AUTH_JWKS_URL", "https://localhost:8090/oauth2/jwks")
	defaultInsecureJWKS := getDefaultInsecureJWKS(authJWKSURL)

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
		Storage: StorageConfig{
			Type:           strings.TrimSpace(getEnvOrDefault("STORAGE_TYPE", "local")),
			LocalBaseDir:   getEnvOrDefault("STORAGE_LOCAL_BASE_DIR", "./bucket"),
			LocalPublicURL: getEnvOrDefault("SERVICE_URL", fmt.Sprintf("http://localhost:%d", serverPort)),
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
		Auth: AuthConfig{
			JWKSURL:               getEnvOrDefault("AUTH_JWKS_URL", "https://localhost:8090/oauth2/jwks"),
			Issuer:                getEnvOrDefault("AUTH_ISSUER", "https://localhost:8090"),
			Audience:              getEnvOrDefault("AUTH_AUDIENCE", "NSW_API"),
			ClientIDs:             parseCommaSeparated(getEnvOrDefault("AUTH_CLIENT_IDS", "TRADER_PORTAL_APP,FCAU_TO_NSW,NPQS_TO_NSW,IRD_TO_NSW")),
			InsecureSkipTLSVerify: getBoolOrDefault("AUTH_JWKS_INSECURE_SKIP_VERIFY", defaultInsecureJWKS),
		},
		Notification: NotificationConfig{
			EmailServiceURL:   os.Getenv("EMAIL_SERVICE_URL"),
			EmailServiceToken: os.Getenv("EMAIL_SERVICE_TOKEN"),

			GovSMSBaseURL:  os.Getenv("GOVSMS_BASE_URL"),
			GovSMSUsername: os.Getenv("GOVSMS_USERNAME"),
			GovSMSPassword: os.Getenv("GOVSMS_PASSWORD"),
			GovSMSSIDCode:  os.Getenv("GOVSMS_SID_CODE"),

			EmailTemplateRoot: getEnvOrDefault("EMAIL_TEMPLATE_ROOT", "./configs/email-templates"),
			SMSTemplateRoot:   getEnvOrDefault("SMS_TEMPLATE_ROOT", "./configs/sms-templates"),
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
	if c.Auth.JWKSURL == "" {
		return fmt.Errorf("AUTH_JWKS_URL is required")
	}
	if c.Auth.Issuer == "" {
		return fmt.Errorf("AUTH_ISSUER is required")
	}
	if c.Auth.Audience == "" {
		return fmt.Errorf("AUTH_AUDIENCE is required")
	}
	if len(c.Auth.ClientIDs) == 0 {
		return fmt.Errorf("AUTH_CLIENT_IDS is required")
	}
	if !c.Server.Debug {
		if len(c.CORS.AllowedOrigins) == 0 {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must be explicitly configured in production")
		}
		for _, origin := range c.CORS.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain '*' in production (SERVER_DEBUG=false)")
			}
		}
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

// getDefaultInsecureJWKS returns true if the JWKS URL is a localhost URL, indicating that TLS verification can be skipped in development
func getDefaultInsecureJWKS(jwksURL string) bool {
	return strings.HasPrefix(jwksURL, "https://localhost") || strings.HasPrefix(jwksURL, "https://127.0.0.1")
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
