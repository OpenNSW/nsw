package store

import (
	"context"

	"github.com/OpenNSW/nsw-task-flow/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTaskStore struct {
	db *gorm.DB
}

func NewGormTaskStore(db *gorm.DB) *GormTaskStore {
	return &GormTaskStore{db: db}
}

func (s *GormTaskStore) SaveTask(ctx context.Context, record store.TaskRecord) {
	model := FromDomain(record)
	// Use Upsert (On Conflict) to handle updates
	s.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&model)
}

func (s *GormTaskStore) GetTask(ctx context.Context, taskID string) (store.TaskRecord, bool) {
	var model TaskRecordModel
	if err := s.db.WithContext(ctx).First(&model, "task_id = ?", taskID).Error; err != nil {
		return store.TaskRecord{}, false
	}
	return model.ToDomain(), true
}

func (s *GormTaskStore) GetTaskByWorkflowID(ctx context.Context, workflowID string) (store.TaskRecord, bool) {
	var model TaskRecordModel
	if err := s.db.WithContext(ctx).First(&model, "task_workflow_id = ?", workflowID).Error; err != nil {
		return store.TaskRecord{}, false
	}
	return model.ToDomain(), true
}

func (s *GormTaskStore) GetAllTasks(ctx context.Context, parentWorkflowID string) []store.TaskRecord {
	var models []TaskRecordModel
	query := s.db.WithContext(ctx)
	if parentWorkflowID != "" {
		query = query.Where("parent_workflow_id = ?", parentWorkflowID)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil
	}

	records := make([]store.TaskRecord, len(models))
	for i, m := range models {
		records[i] = m.ToDomain()
	}
	return records
}
