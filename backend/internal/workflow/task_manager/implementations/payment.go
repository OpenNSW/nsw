package implementations

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

type PaymentTask struct {
	BaseTask
}

func (t *PaymentTask) CanExecute(ctx context.Context, taskCtx *task_manager.TaskContext) (bool, error) {
	// Payment tasks can execute when status is READY
	return t.TaskModel.Status == model.TaskStatusReady, nil
}

func (t *PaymentTask) Execute(ctx context.Context, taskCtx *task_manager.TaskContext) (*task_manager.TaskResult, error) {
	// Handle payment processing
	// Payment gateway integration will be added in later PR
	return &task_manager.TaskResult{
		Status:  model.TaskStatusSubmitted,
		Message: "Payment processed successfully",
	}, nil
}
