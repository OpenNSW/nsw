package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// BaseTask provides common functionality for all task types
type BaseTask struct {
	ID        uuid.UUID
	TaskType  task.TaskType
	TaskModel *model.Task
}

func (b *BaseTask) GetID() uuid.UUID {
	return b.ID
}

func (b *BaseTask) GetType() task.TaskType {
	return b.TaskType
}

func (b *BaseTask) CanExecute(ctx context.Context, taskCtx *task.TaskContext) (bool, error) {
	return b.TaskModel.Status == model.TaskStatusReady, nil
}
