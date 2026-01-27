package oga

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// JSONB is a custom type for storing JSON data in SQLite
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, j)
}

// ApplicationRecord represents an application in the OGA database
type ApplicationRecord struct {
	TaskID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ConsignmentID uuid.UUID  `gorm:"type:uuid;index;not null"`
	ServiceURL    string     `gorm:"type:varchar(512);not null"`                  // URL to send response back to
	Data          JSONB      `gorm:"type:text"`                                   // Injected data from service
	Status        string     `gorm:"type:varchar(50);not null;default:'PENDING'"` // PENDING, APPROVED, REJECTED
	ReviewerNotes string     `gorm:"type:text"`                                   // Optional notes from reviewer
	ReviewedAt    *time.Time `gorm:"type:datetime"`                               // When it was reviewed
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
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
	if err := s.db.Where("status = ?", status).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// UpdateStatus updates the status of an application
func (s *ApplicationStore) UpdateStatus(taskID uuid.UUID, status string, reviewerNotes string) error {
	now := time.Now()
	return s.db.Model(&ApplicationRecord{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":         status,
			"reviewer_notes": reviewerNotes,
			"reviewed_at":    now,
			"updated_at":     now,
		}).Error
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
