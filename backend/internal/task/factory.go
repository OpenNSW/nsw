package task

import (
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// taskFactory implements TaskFactory interface
type taskFactory struct{}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory() TaskFactory {
	return &taskFactory{}
}

func (f *taskFactory) CreateTask(taskType TaskType, taskModel *model.Task) (Task, error) {
	baseTask := BaseTask{
		ID:       taskModel.ID,
		TaskType: taskType,
	}

	switch taskType {
	case TaskTypeTraderForm:
		return &TraderFormTask{BaseTask: baseTask}, nil
	case TaskTypeOGAForm:
		return &OGAFormTask{BaseTask: baseTask}, nil
	case TaskTypeWaitForEvent:
		return &WaitForEventTask{BaseTask: baseTask}, nil
	case TaskTypePayment:
		return &PaymentTask{BaseTask: baseTask}, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
