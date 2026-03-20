package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresConnector implements DBConnector for PostgreSQL.
type PostgresConnector struct {
	Host, Port, User, Password, Name, SSLMode string
}

// Open establishes a connection to the PostgreSQL database.
func (c *PostgresConnector) Open() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
