package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type PaymentTask struct {
	BaseTask
}

func (t *PaymentTask) Execute(ctx context.Context, taskCtx *task.TaskContext) (*task.TaskResult, error) {
	// Handle payment processing
	// Payment gateway integration will be added in later PR
	return &task.TaskResult{
		Status:  model.TaskStatusSubmitted,
		Message: "Payment processed successfully",
	}, nil
}
