package task_manager

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
	CanExecute(ctx context.Context, taskCtx *TaskContext) (bool, error)
}

// TaskResult represents the outcome of task execution
type TaskResult struct {
	Status    model.TaskStatus `json:"status"`
	Message   string           `json:"message,omitempty"`
	Data      interface{}      `json:"data,omitempty"`      // Task-specific result data
	NextTasks []uuid.UUID      `json:"nextTasks,omitempty"` // Tasks to activate next
}

// TaskContext provides context for task execution
type TaskContext struct {
	TaskID        uuid.UUID
	ConsignmentID uuid.UUID
	AssigneeID    uuid.UUID
	FormTemplate  *model.FormTemplate
	FormData      map[string]interface{} // Submitted form data
	Metadata      map[string]interface{} // Additional context
}
