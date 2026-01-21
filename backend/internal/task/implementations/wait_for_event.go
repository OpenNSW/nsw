package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type WaitForEventTask struct {
	BaseTask
}

func (t *WaitForEventTask) Execute(ctx context.Context, taskCtx *task.TaskContext) (*task.TaskResult, error) {
	// Wait for external event/callback
	// This task will be completed when the event is received via NotifyTaskCompletion (handled in later PR)
	// Status is set to SUBMITTED to prevent re-execution (READY would cause busy-loop)
	return &task.TaskResult{
		Status:  model.TaskStatusSubmitted,
		Message: "Waiting for external event",
	}, nil
}
