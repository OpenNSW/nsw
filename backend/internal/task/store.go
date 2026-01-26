package task

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TaskRecord represents a task execution record in the database
type TaskRecord struct {
	ID            uuid.UUID        `gorm:"type:uuid;primaryKey"`
	StepID        string           `gorm:"type:varchar(50);not null"`
	ConsignmentID uuid.UUID        `gorm:"type:uuid;index;not null"`
	Type          Type             `gorm:"type:varchar(50);not null"`
	Status        model.TaskStatus `gorm:"type:varchar(50);not null"`
	CommandSet    json.RawMessage  `gorm:"type:json"`
	CreatedAt     time.Time        `gorm:"autoCreateTime"`
	UpdatedAt     time.Time        `gorm:"autoUpdateTime"`
}

// TableName returns the table name for TaskExecution
func (TaskRecord) TableName() string {
	return "task_executions"
}

// TaskStore handles database operations for task executions
type TaskStore struct {
	db *gorm.DB
}

// NewTaskStore creates a new TaskStore with SQLite database
func NewTaskStore(dbPath string) (*TaskStore, error) {
	if dbPath == "" {
		dbPath = "task_executions.db"
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&TaskRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &TaskStore{db: db}, nil
}

// NewInMemoryTaskStore creates a TaskStore with in-memory SQLite database (useful for testing)
func NewInMemoryTaskStore() (*TaskStore, error) {
	return NewTaskStore(":memory:")
}

// Create inserts a new task execution record
func (s *TaskStore) Create(execution *TaskRecord) error {
	return s.db.Create(execution).Error
}

// GetByID retrieves a task execution by its ID
func (s *TaskStore) GetByID(id uuid.UUID) (*TaskRecord, error) {
	var taskRecord TaskRecord
	if err := s.db.First(&taskRecord, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &taskRecord, nil
}

// UpdateStatus updates the status of a task execution
func (s *TaskStore) UpdateStatus(id uuid.UUID, status model.TaskStatus) error {
	return s.db.Model(&TaskRecord{}).Where("id = ?", id).Update("status", status).Error
}

// Update updates a task execution record
func (s *TaskStore) Update(execution *TaskRecord) error {
	return s.db.Save(execution).Error
}

// Delete removes a task execution record
func (s *TaskStore) Delete(id uuid.UUID) error {
	return s.db.Delete(&TaskRecord{}, "id = ?", id).Error
}

// GetAll retrieves all task executions
func (s *TaskStore) GetAll() ([]TaskRecord, error) {
	var executions []TaskRecord
	if err := s.db.Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}

// GetByStatus retrieves task executions by status
func (s *TaskStore) GetByStatus(status model.TaskStatus) ([]TaskRecord, error) {
	var executions []TaskRecord
	if err := s.db.Where("status = ?", status).Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}

// Close closes the database connection
func (s *TaskStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
