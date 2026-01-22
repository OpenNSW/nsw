package task

import (
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// BaseTask provides common functionality for all task types
type BaseTask struct {
	ID         uuid.UUID
	TaskType   TaskType
	TaskStatus model.TaskStatus
}

func (b *BaseTask) GetID() uuid.UUID {
	return b.ID
}

func (b *BaseTask) GetType() TaskType {
	return b.TaskType
}

func (b *BaseTask) CanExecute() (bool, error) {
	return b.TaskStatus == model.TaskStatusReady, nil
}
