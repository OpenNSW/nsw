package task

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskInfo represents a task execution record in the database
type TaskInfo struct {
	ID            uuid.UUID        `gorm:"type:uuid;primaryKey"`
	StepID        string           `gorm:"type:varchar(50);not null"`
	ConsignmentID uuid.UUID        `gorm:"type:uuid;index;not null"`
	Type          Type             `gorm:"type:varchar(50);not null"`
	Status        model.TaskStatus `gorm:"type:varchar(50);not null"`
	CommandSet    json.RawMessage  `gorm:"type:json"`
	GlobalContext json.RawMessage  `gorm:"type:json"`
	CreatedAt     time.Time        `gorm:"autoCreateTime"`
	UpdatedAt     time.Time        `gorm:"autoUpdateTime"`
}

// TableName returns the table name for TaskInfo
func (TaskInfo) TableName() string {
	return "task_infos"
}

// TaskStore handles database operations for task infos
type TaskStore struct {
	db *gorm.DB
}

// NewTaskStore creates a new TaskStore with the provided database connection
func NewTaskStore(db *gorm.DB) (*TaskStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	return &TaskStore{db: db}, nil
}

// Create inserts a new task execution record
func (s *TaskStore) Create(execution *TaskInfo) error {
	return s.db.Create(execution).Error
}

// GetByID retrieves a task execution by its ID
func (s *TaskStore) GetByID(id uuid.UUID) (*TaskInfo, error) {
	var taskRecord TaskInfo
	if err := s.db.First(&taskRecord, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &taskRecord, nil
}

// UpdateStatus updates the status of a task execution
func (s *TaskStore) UpdateStatus(id uuid.UUID, status model.TaskStatus) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("status", status).Error
}

// Update updates a task execution record
func (s *TaskStore) Update(execution *TaskInfo) error {
	return s.db.Save(execution).Error
}

// Delete removes a task execution record
func (s *TaskStore) Delete(id uuid.UUID) error {
	return s.db.Delete(&TaskInfo{}, "id = ?", id).Error
}

// GetAll retrieves all task executions
func (s *TaskStore) GetAll() ([]TaskInfo, error) {
	var executions []TaskInfo
	if err := s.db.Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}

// GetByStatus retrieves task executions by status
func (s *TaskStore) GetByStatus(status model.TaskStatus) ([]TaskInfo, error) {
	var executions []TaskInfo
	if err := s.db.Where("status = ?", status).Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}
