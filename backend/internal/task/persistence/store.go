package persistence

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskInfo represents a task execution record in the database
type TaskInfo struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey"`
	StepID        string          `gorm:"type:varchar(50);not null"`
	ConsignmentID uuid.UUID       `gorm:"type:uuid;index;not null"`
	Type          plugin.Type     `gorm:"type:varchar(50);not null"`
	State         plugin.State    `gorm:"type:varchar(50);not null"` // Container-level state (lifecycle)
	PluginState   string          `gorm:"type:varchar(100)"`         // Plugin-level state (business logic)
	Config        json.RawMessage `gorm:"type:json"`
	LocalState    json.RawMessage `gorm:"type:json"`
	GlobalContext json.RawMessage `gorm:"type:json"`
	CreatedAt     time.Time       `gorm:"autoCreateTime"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime"`
}

// TableName returns the table name for TaskInfo
func (TaskInfo) TableName() string {
	return "task_infos"
}

// TaskStore handles database operations for task infos
type TaskStore struct {
	db *gorm.DB
}

type TaskStoreInterface interface {
	Create(*TaskInfo) error
	GetByID(uuid.UUID) (*TaskInfo, error)
	UpdateStatus(uuid.UUID, *plugin.State) error
	Update(*TaskInfo) error
	Delete(uuid.UUID) error
	GetAll() ([]TaskInfo, error)
	GetByStatus(plugin.State) ([]TaskInfo, error)
	UpdateLocalState(uuid.UUID, json.RawMessage) error
	GetLocalState(uuid.UUID) (json.RawMessage, error)
	UpdatePluginState(uuid.UUID, string) error
	GetPluginState(uuid.UUID) (string, error)
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
func (s *TaskStore) UpdateStatus(id uuid.UUID, status *plugin.State) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("state", &status).Error
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
func (s *TaskStore) GetByStatus(status plugin.State) ([]TaskInfo, error) {
	var executions []TaskInfo
	if err := s.db.Where("status = ?", status).Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}

// UpdateLocalState updates the local state of a task execution
func (s *TaskStore) UpdateLocalState(id uuid.UUID, localState json.RawMessage) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("local_state", localState).Error
}

// GetLocalState retrieves the local state of a task execution
func (s *TaskStore) GetLocalState(id uuid.UUID) (json.RawMessage, error) {
	var taskInfo TaskInfo
	if err := s.db.Select("local_state").First(&taskInfo, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return taskInfo.LocalState, nil
}

// UpdatePluginState updates the plugin state of a task execution
func (s *TaskStore) UpdatePluginState(id uuid.UUID, pluginState string) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("plugin_state", pluginState).Error
}

// GetPluginState retrieves the plugin state of a task execution
func (s *TaskStore) GetPluginState(id uuid.UUID) (string, error) {
	var taskInfo TaskInfo
	if err := s.db.Select("plugin_state").First(&taskInfo, "id = ?", id).Error; err != nil {
		return "", err
	}
	return taskInfo.PluginState, nil
}

// Close closes the database connection
func (s *TaskStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
