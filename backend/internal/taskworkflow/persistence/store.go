package persistence

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/task/plugin"
)

// TaskWorkflowTask represents one persisted task within a task workflow.
type TaskWorkflowTask struct {
	TaskID          string          `gorm:"type:text;column:task_id;not null;primaryKey" json:"taskId"`
	MacroWorkflowID string          `gorm:"type:text;column:macro_workflow_id;not null;index" json:"macroWorkflowId"`
	TaskTemplateID  string          `gorm:"type:text;column:task_template_id;not null;index" json:"taskTemplateId"`
	State           plugin.State    `gorm:"type:varchar(50);column:state;not null;index" json:"state"`
	Data            json.RawMessage `gorm:"type:jsonb;column:data;serializer:json;not null" json:"data"`
	CreatedAt       time.Time       `gorm:"type:timestamptz;column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time       `gorm:"type:timestamptz;column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}

func (TaskWorkflowTask) TableName() string {
	return "task_workflow_tasks"
}

// Store is the full task workflow task repository.
//
// Use this from orchestration/runtime code that needs lifecycle-level access:
// creating task rows, looking up tasks across a macro workflow, or deleting task
// records. Do not pass Store to subtasks or renderers; use TaskScopedStore for
// task-local access.
type Store interface {
	Create(*TaskWorkflowTask) error
	GetByTaskID(string) (*TaskWorkflowTask, error)
	GetStateByTaskID(string) (plugin.State, error)
	GetDataByTaskID(string) (json.RawMessage, error)
	GetByMacroWorkflowID(string) ([]TaskWorkflowTask, error)
	Update(*TaskWorkflowTask) error
	UpdateState(string, plugin.State) error
	UpdateData(string, json.RawMessage) error
	Delete(string) error
}

type TaskWorkflowStore struct {
	db *gorm.DB
}

func NewTaskWorkflowStore(db *gorm.DB) (*TaskWorkflowStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	return &TaskWorkflowStore{db: db}, nil
}

func (s *TaskWorkflowStore) Create(task *TaskWorkflowTask) error {
	return s.db.Create(task).Error
}

func (s *TaskWorkflowStore) GetByTaskID(taskID string) (*TaskWorkflowTask, error) {
	var task TaskWorkflowTask
	if err := s.db.First(&task, "task_id = ?", taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskWorkflowStore) GetStateByTaskID(taskID string) (plugin.State, error) {
	var task TaskWorkflowTask
	if err := s.db.Select("state").First(&task, "task_id = ?", taskID).Error; err != nil {
		return "", err
	}
	return task.State, nil
}

func (s *TaskWorkflowStore) GetDataByTaskID(taskID string) (json.RawMessage, error) {
	var task TaskWorkflowTask
	if err := s.db.Select("data").First(&task, "task_id = ?", taskID).Error; err != nil {
		return nil, err
	}
	return task.Data, nil
}

func (s *TaskWorkflowStore) GetByMacroWorkflowID(macroWorkflowID string) ([]TaskWorkflowTask, error) {
	var tasks []TaskWorkflowTask
	if err := s.db.Where("macro_workflow_id = ?", macroWorkflowID).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskWorkflowStore) Update(task *TaskWorkflowTask) error {
	result := s.db.Model(&TaskWorkflowTask{}).
		Where("task_id = ?", task.TaskID).
		Select("*").
		Updates(task)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *TaskWorkflowStore) UpdateState(taskID string, state plugin.State) error {
	return s.db.Model(&TaskWorkflowTask{}).Where("task_id = ?", taskID).Update("state", state).Error
}

func (s *TaskWorkflowStore) UpdateData(taskID string, data json.RawMessage) error {
	return s.db.Model(&TaskWorkflowTask{}).Where("task_id = ?", taskID).Update("data", data).Error
}

func (s *TaskWorkflowStore) Delete(taskID string) error {
	result := s.db.Delete(&TaskWorkflowTask{}, "task_id = ?", taskID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
