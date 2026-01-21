package implementations

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type TraderFormTask struct {
	BaseTask
}

func (t *TraderFormTask) Execute(ctx context.Context, taskCtx *task.TaskContext) (*task.TaskResult, error) {
	// This method is called when Trader Portal submits the form via SubmitTaskCompletion
	// 1. Get form data from context (set by SubmitTaskCompletion)
	formData, ok := ctx.Value("formData").(map[string]interface{})
	if !ok || formData == nil || len(formData) == 0 {
		return nil, fmt.Errorf("form data is required for trader form submission")
	}

	// Form template can be accessed from taskCtx.Task.TraderFormTemplate if needed

	// 2. Save form submission to database (using transaction from context)
	// TODO: Create FormSubmission record using taskCtx.Tx
	// For now, we'll just validate and return success

	// 3. Update task status to SUBMITTED
	// The status update is handled by TaskManager in the transaction

	return &task.TaskResult{
		Status:  model.TaskStatusSubmitted,
		Message: "Trader form submitted successfully",
	}, nil
}
