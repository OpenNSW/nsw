package task

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type WaitForEventTask struct {
	CommandSet interface{}
}

func (t *WaitForEventTask) Execute(_ context.Context, payload interface{}) (*ExecutionResult, error) {
	// Wait for external event/callback
	// This task will be completed when the event is received via NotifyTaskCompletion (handled in later PR)
	// Status is set to SUBMITTED to prevent re-execution (READY would cause busy-loop)
	return &ExecutionResult{
		Status:  model.TaskStatusInProgress,
		Message: "Waiting for external event",
	}, nil
}
