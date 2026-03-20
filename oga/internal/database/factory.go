package database

import "fmt"

// Config holds the database configuration needed by the factory.
type Config struct {
	Driver   string // "sqlite" or "postgres"
	Path     string // SQLite file path
	Host     string // PostgreSQL host
	Port     string // PostgreSQL port
	User     string // PostgreSQL user
	Password string // PostgreSQL password
	Name     string // PostgreSQL database name
	SSLMode  string // PostgreSQL SSL mode
}

// NewConnector creates a new DBConnector based on the configuration driver.
func NewConnector(cfg Config) (DBConnector, error) {
	switch cfg.Driver {
	case "sqlite":
		return &SQLiteConnector{Path: cfg.Path}, nil
	case "postgres":
		return &PostgresConnector{
			Host:     cfg.Host,
			Port:     cfg.Port,
			User:     cfg.User,
			Password: cfg.Password,
			Name:     cfg.Name,
			SSLMode:  cfg.SSLMode,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}
