package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type OGAFormTask struct {
	BaseTask
	ExternalAPIURL string // Will be loaded from config in later PR
}

func (t *OGAFormTask) Execute(ctx context.Context, taskCtx *task.TaskContext) (*task.TaskResult, error) {
	// This method is called for non-realtime OGA tasks
	// 1. Route to external OGA system (AYUSCUDA, etc.)
	// TODO: Implement actual HTTP call to external OGA API
	// For now, we'll simulate the routing

	// 2. Update task status to IN_PROGRESS (will be updated to COMPLETED/REJECTED when OGA notifies)
	// The actual status update happens when OGA calls NotifyTaskCompletion

	// 3. Return IN_PROGRESS status
	// Task Manager will notify Workflow Manager with INPROGRESS state
	return &task.TaskResult{
		Status:  model.TaskStatusSubmitted, // Submitted to external system, waiting for OGA response
		Message: "OGA form routed to external system",
	}, nil
}
