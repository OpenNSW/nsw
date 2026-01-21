package implementations

import (
	"context"

	"github.com/google/uuid"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

type TraderFormTask struct {
	BaseTask
}

func (t *TraderFormTask) CanExecute(ctx context.Context, taskCtx *task_manager.TaskContext) (bool, error) {
	// Trader form tasks can execute when status is READY
	return t.TaskModel.Status == model.TaskStatusReady, nil
}

func (t *TraderFormTask) Execute(ctx context.Context, taskCtx *task_manager.TaskContext) (*task_manager.TaskResult, error) {
	// 1. Validate form data
	// 2. Save form submission to database
	// 3. Update task status to SUBMITTED
	// 4. Return result with next tasks to activate

	return &task_manager.TaskResult{
		Status:    model.TaskStatusSubmitted,
		Message:   "Trader form submitted successfully",
		NextTasks: []uuid.UUID{}, // Will be determined by Workflow Engine
	}, nil
}
