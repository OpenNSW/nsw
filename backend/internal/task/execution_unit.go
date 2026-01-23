package task

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// ExecutionUnit represents a unit of work in the workflow system
type ExecutionUnit interface {
	// Execute performs the task's work and returns the result
	Execute(ctx context.Context, payload interface{}) (*ExecutionResult, error)
}

// ExecutionResult represents the outcome of task execution
type ExecutionResult struct {
	Status  model.TaskStatus `json:"status"`
	Message string           `json:"message,omitempty"`
	Data    interface{}      `json:"data,omitempty"` // Additional data specific to the task type
}
