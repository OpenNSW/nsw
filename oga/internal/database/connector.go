package database

import "gorm.io/gorm"

// DBConnector abstracts the driver-specific logic for opening a GORM connection.
type DBConnector interface {
	Open() (*gorm.DB, error)
}
