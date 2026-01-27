package oga

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ApplicationRecord represents an application in the OGA database
type ApplicationRecord struct {
	TaskID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	ConsignmentID uuid.UUID `gorm:"type:uuid;index;not null"`
	StepID        string    `gorm:"type:varchar(255);not null"`
	FormID        string    `gorm:"type:varchar(255);not null"`
	Status        string    `gorm:"type:varchar(50);not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for ApplicationRecord
func (ApplicationRecord) TableName() string {
	return "applications"
}

// ApplicationStore handles database operations for OGA applications
type ApplicationStore struct {
	db *gorm.DB
}

// NewApplicationStore creates a new ApplicationStore with SQLite database
func NewApplicationStore(dbPath string) (*ApplicationStore, error) {
	if dbPath == "" {
		dbPath = "oga_applications.db"
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&ApplicationRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &ApplicationStore{db: db}, nil
}

// CreateOrUpdate creates or updates an application record
func (s *ApplicationStore) CreateOrUpdate(app *ApplicationRecord) error {
	return s.db.Save(app).Error
}

// GetByTaskID retrieves an application by task ID
func (s *ApplicationStore) GetByTaskID(taskID uuid.UUID) (*ApplicationRecord, error) {
	var app ApplicationRecord
	if err := s.db.First(&app, "task_id = ?", taskID).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// GetAll retrieves all applications
func (s *ApplicationStore) GetAll() ([]ApplicationRecord, error) {
	var apps []ApplicationRecord
	if err := s.db.Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// GetByStatus retrieves applications by status
func (s *ApplicationStore) GetByStatus(status string) ([]ApplicationRecord, error) {
	var apps []ApplicationRecord
	if err := s.db.Where("status = ?", status).Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// Delete removes an application by task ID
func (s *ApplicationStore) Delete(taskID uuid.UUID) error {
	return s.db.Delete(&ApplicationRecord{}, "task_id = ?", taskID).Error
}

// Close closes the database connection
func (s *ApplicationStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
