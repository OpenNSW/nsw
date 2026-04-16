package persistence

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/task/plugin"
)

// TaskInfo represents a task execution record in the database
type TaskInfo struct {
	ID                     string          `gorm:"type:text;column:id;not null;primaryKey" json:"id"`
	WorkflowID             string          `gorm:"type:text;column:workflow_id;not null;index" json:"workflowId"`
	WorkflowNodeTemplateID string          `gorm:"type:text;column:workflow_node_template_id;not null" json:"workflowNodeTemplateId"`
	Type                   plugin.Type     `gorm:"type:varchar(50);column:type;not null" json:"type"`
	State                  plugin.State    `gorm:"type:varchar(50);column:state;not null" json:"state"`      // Container-level state (lifecycle)
	PluginState            string          `gorm:"type:varchar(100);column:plugin_state" json:"pluginState"` // Plugin-level state (business logic)
	Config                 json.RawMessage `gorm:"type:jsonb;column:config;serializer:json" json:"config"`
	LocalState             json.RawMessage `gorm:"type:jsonb;column:local_state;serializer:json" json:"localState"`
	GlobalContext          json.RawMessage `gorm:"type:jsonb;column:global_context;serializer:json" json:"globalContext"`
	CreatedAt              time.Time       `gorm:"type:timestamptz;column:created_at;not null" json:"createdAt"`
	UpdatedAt              time.Time       `gorm:"type:timestamptz;column:updated_at;not null" json:"updatedAt"`
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
	GetByID(string) (*TaskInfo, error)
	CheckOwnership(string, string) (bool, error)
	UpdateStatus(string, *plugin.State) error
	Update(*TaskInfo) error
	Delete(string) error
	GetAll() ([]TaskInfo, error)
	GetByStatus(plugin.State) ([]TaskInfo, error)
	UpdateLocalState(string, json.RawMessage) error
	GetLocalState(string) (json.RawMessage, error)
	UpdatePluginState(string, string) error
	GetPluginState(string) (string, error)
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
func (s *TaskStore) GetByID(id string) (*TaskInfo, error) {
	var taskRecord TaskInfo
	if err := s.db.First(&taskRecord, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &taskRecord, nil
}

// CheckOwnership verifies if a task belongs to a consignment owned by the requestor
func (s *TaskStore) CheckOwnership(id string, requestorID string) (bool, error) {
	var count int64
	err := s.db.Table("task_infos").
		Joins("JOIN consignments ON task_infos.workflow_id = consignments.id").
		Where("task_infos.id = ?", id).
		Where("(consignments.trader_id = ? OR consignments.cha_id = ?)", requestorID, requestorID).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateStatus updates the status of a task execution
func (s *TaskStore) UpdateStatus(id string, status *plugin.State) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("state", &status).Error
}

// Update updates a task execution record
func (s *TaskStore) Update(execution *TaskInfo) error {
	return s.db.Save(execution).Error
}

// Delete removes a task execution record
func (s *TaskStore) Delete(id string) error {
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
func (s *TaskStore) UpdateLocalState(id string, localState json.RawMessage) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("local_state", localState).Error
}

// GetLocalState retrieves the local state of a task execution
func (s *TaskStore) GetLocalState(id string) (json.RawMessage, error) {
	var taskInfo TaskInfo
	if err := s.db.Select("local_state").First(&taskInfo, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return taskInfo.LocalState, nil
}

// UpdatePluginState updates the plugin state of a task execution
func (s *TaskStore) UpdatePluginState(id string, pluginState string) error {
	return s.db.Model(&TaskInfo{}).Where("id = ?", id).Update("plugin_state", pluginState).Error
}

// GetPluginState retrieves the plugin state of a task execution
func (s *TaskStore) GetPluginState(id string) (string, error) {
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
