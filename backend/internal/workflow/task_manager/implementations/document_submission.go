package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

type DocumentSubmissionTask struct {
	BaseTask
}

func (t *DocumentSubmissionTask) CanExecute(ctx context.Context, taskCtx *task_manager.TaskContext) (bool, error) {
	// Document submission tasks can execute when status is READY
	return t.TaskModel.Status == model.TaskStatusReady, nil
}

func (t *DocumentSubmissionTask) Execute(ctx context.Context, taskCtx *task_manager.TaskContext) (*task_manager.TaskResult, error) {
	// Handle document uploads
	// Documents are stored in FormSubmission.Data as file references
	return &task_manager.TaskResult{
		Status:  model.TaskStatusSubmitted,
		Message: "Documents submitted successfully",
	}, nil
}
