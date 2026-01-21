package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

type WaitForEventTask struct {
	BaseTask
}

func (t *WaitForEventTask) CanExecute(ctx context.Context, taskCtx *task_manager.TaskContext) (bool, error) {
	// Wait for event tasks can execute when status is READY
	return t.TaskModel.Status == model.TaskStatusReady, nil
}

func (t *WaitForEventTask) Execute(ctx context.Context, taskCtx *task_manager.TaskContext) (*task_manager.TaskResult, error) {
	// Wait for external event/callback
	// This task will be completed when the event is received (handled in later PR)
	return &task_manager.TaskResult{
		Status:  model.TaskStatusReady,
		Message: "Waiting for external event",
	}, nil
}
