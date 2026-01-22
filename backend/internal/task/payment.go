package task

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type PaymentTask struct {
	BaseTask
}

func (t *PaymentTask) Execute(_ context.Context, _ *TaskContext) (*TaskResult, error) {
	// Handle payment processing
	// Payment gateway integration will be added in later PR
	return &TaskResult{
		Status:  model.TaskStatusInProgress,
		Message: "Payment processed successfully",
	}, nil
}
