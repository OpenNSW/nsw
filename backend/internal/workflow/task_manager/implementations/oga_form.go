package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

type OGAFormTask struct {
	BaseTask
	ExternalAPIURL string // Will be loaded from config in later PR
}

func (t *OGAFormTask) CanExecute(ctx context.Context, taskCtx *task_manager.TaskContext) (bool, error) {
	// OGA form tasks can execute when status is READY
	return t.TaskModel.Status == model.TaskStatusReady, nil
}

func (t *OGAFormTask) Execute(ctx context.Context, taskCtx *task_manager.TaskContext) (*task_manager.TaskResult, error) {
	// 1. Validate form data
	// 2. If real-time: Save to database and notify Workflow Engine
	// 3. If offline: Route to external OGA system (AYUSCUDA, etc.)
	// 4. Update task status

	// For this PR: Just save to database
	return &task_manager.TaskResult{
		Status:  model.TaskStatusSubmitted,
		Message: "OGA form submitted",
	}, nil
}
