package task

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// Task represents a unit of work in the workflow system
type Task interface {
	// Execute performs the task's work and returns the result
	Execute(ctx context.Context, taskCtx *TaskContext) (*TaskResult, error)

	// GetType returns the type of this task
	GetType() TaskType

	// GetID returns the unique identifier for this task
	GetID() uuid.UUID

	// CanExecute checks if the task is ready to be executed
	CanExecute() (bool, error)
}

// TaskResult represents the outcome of task execution
type TaskResult struct {
	Status  model.TaskStatus `json:"status"`
	Message string           `json:"message,omitempty"`
}

// TaskContext provides context for task execution
type TaskContext struct {
	Task          *model.Task // Full task model with all task-level information
	ConsignmentID uuid.UUID
}
