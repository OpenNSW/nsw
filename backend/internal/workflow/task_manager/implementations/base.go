package implementations

import (
	"github.com/google/uuid"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

// BaseTask provides common functionality for all task types
type BaseTask struct {
	ID        uuid.UUID
	TaskType  task_manager.TaskType
	TaskModel *model.Task
}

func (b *BaseTask) GetID() uuid.UUID {
	return b.ID
}

func (b *BaseTask) GetType() task_manager.TaskType {
	return b.TaskType
}
