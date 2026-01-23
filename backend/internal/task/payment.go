package task

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type PaymentTask struct {
	CommandSet interface{}
}

func (t *PaymentTask) Execute(_ context.Context, payload interface{}) (*ExecutionResult, error) {
	// Handle payment processing
	// Payment gateway integration will be added in later PR
	return &ExecutionResult{
		Status:  model.TaskStatusInProgress,
		Message: "Payment processed successfully",
	}, nil
}
