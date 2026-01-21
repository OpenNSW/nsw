package task

import (
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeTraderForm   TaskType = "TRADER_FORM"
	TaskTypeOGAForm      TaskType = "OGA_FORM"
	TaskTypeWaitForEvent TaskType = "WAIT_FOR_EVENT"
	TaskTypePayment      TaskType = "PAYMENT"
)

// TaskFactory creates task instances from task type and model
type TaskFactory interface {
	CreateTask(taskType TaskType, taskModel *model.Task) (Task, error)
}
