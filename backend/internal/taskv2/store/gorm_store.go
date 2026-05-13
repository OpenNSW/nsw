// Package store provides a GORM/Postgres implementation of the
// nsw-task-flow store.TaskStore interface, backed by the
// task_workflow_tasks table (see migration 019).
package store

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"time"

	tfstore "github.com/OpenNSW/nsw-task-flow/store"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// jsonbBytes carries the raw JSON payload for the task_workflow_tasks.data
// JSONB column. Using a local type keeps us off gorm.io/datatypes (which drags
// in a MySQL driver chain we don't use).
type jsonbBytes []byte

func (j jsonbBytes) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *jsonbBytes) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = []byte(v)
	default:
		return fmt.Errorf("jsonbBytes.Scan: unsupported type %T", src)
	}
	return nil
}

// taskRow is the GORM model for the task_workflow_tasks table.
// It mirrors store.TaskRecord, with the JSONB Data column persisted as raw bytes.
type taskRow struct {
	TaskID               string     `gorm:"column:task_id;primaryKey"`
	TaskType             string     `gorm:"column:task_type"`
	UserFormID           string     `gorm:"column:user_form_id"`
	ReviewerFormID       string     `gorm:"column:reviewer_form_id"`
	Status               string     `gorm:"column:status"`
	ParentWorkflowID     string     `gorm:"column:parent_workflow_id;index"`
	ParentRunID          string     `gorm:"column:parent_run_id"`
	ParentNodeID         string     `gorm:"column:parent_node_id"`
	TaskWorkflowID       string     `gorm:"column:task_workflow_id;index"`
	TaskRunID            string     `gorm:"column:task_run_id"`
	SubTaskNodeID        string     `gorm:"column:subtask_node_id"`
	ActiveTaskTemplateID string     `gorm:"column:active_task_template_id"`
	Data                 jsonbBytes `gorm:"column:data;type:jsonb;not null"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (taskRow) TableName() string { return "task_workflow_tasks" }

// GormTaskStore implements nsw-task-flow's store.TaskStore on top of GORM/Postgres.
type GormTaskStore struct {
	db *gorm.DB
}

// NewGormTaskStore returns a TaskStore backed by the given GORM handle.
func NewGormTaskStore(db *gorm.DB) *GormTaskStore {
	return &GormTaskStore{db: db}
}

// SaveTask upserts a TaskRecord. Errors are logged — the TaskStore interface
// is fire-and-forget, matching the demo's in-memory implementation.
func (s *GormTaskStore) SaveTask(record tfstore.TaskRecord) {
	row, err := toRow(record)
	if err != nil {
		slog.Error("taskv2 store: failed to marshal record", "taskId", record.TaskID, "error", err)
		return
	}
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_id"}},
		UpdateAll: true,
	}).Create(&row).Error; err != nil {
		slog.Error("taskv2 store: failed to upsert task", "taskId", record.TaskID, "error", err)
	}
}

// GetTask returns the record with the given ID, or false if missing.
func (s *GormTaskStore) GetTask(taskID string) (tfstore.TaskRecord, bool) {
	var row taskRow
	if err := s.db.First(&row, "task_id = ?", taskID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("taskv2 store: get by id failed", "taskId", taskID, "error", err)
		}
		return tfstore.TaskRecord{}, false
	}
	rec, err := fromRow(row)
	if err != nil {
		slog.Error("taskv2 store: unmarshal failed", "taskId", taskID, "error", err)
		return tfstore.TaskRecord{}, false
	}
	return rec, true
}

// GetTaskByWorkflowID looks up by task_workflow_id — used by the task workflow
// completion handler to resolve which TaskRecord owns an incoming completion.
func (s *GormTaskStore) GetTaskByWorkflowID(workflowID string) (tfstore.TaskRecord, bool) {
	var row taskRow
	if err := s.db.First(&row, "task_workflow_id = ?", workflowID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("taskv2 store: get by workflow id failed", "workflowId", workflowID, "error", err)
		}
		return tfstore.TaskRecord{}, false
	}
	rec, err := fromRow(row)
	if err != nil {
		slog.Error("taskv2 store: unmarshal failed", "workflowId", workflowID, "error", err)
		return tfstore.TaskRecord{}, false
	}
	return rec, true
}

// GetAllTasks returns every TaskRecord, ordered by creation time ascending so
// list views render in stable order.
func (s *GormTaskStore) GetAllTasks() []tfstore.TaskRecord {
	var rows []taskRow
	if err := s.db.Order("created_at ASC").Find(&rows).Error; err != nil {
		slog.Error("taskv2 store: list failed", "error", err)
		return nil
	}
	out := make([]tfstore.TaskRecord, 0, len(rows))
	for _, row := range rows {
		rec, err := fromRow(row)
		if err != nil {
			slog.Error("taskv2 store: unmarshal in list failed", "taskId", row.TaskID, "error", err)
			continue
		}
		out = append(out, rec)
	}
	return out
}
