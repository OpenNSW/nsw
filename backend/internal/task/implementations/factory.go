package implementations

import (
	"fmt"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// taskFactory implements TaskFactory interface
type taskFactory struct{}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory() task.TaskFactory {
	return &taskFactory{}
}

func (f *taskFactory) CreateTask(taskType task.TaskType, taskModel *model.Task) (task.Task, error) {
	baseTask := BaseTask{
		ID:        taskModel.ID,
		TaskType:  taskType,
		TaskModel: taskModel,
	}

	switch taskType {
	case task.TaskTypeTraderForm:
		return &TraderFormTask{BaseTask: baseTask}, nil
	case task.TaskTypeOGAForm:
		return &OGAFormTask{BaseTask: baseTask}, nil
	case task.TaskTypeWaitForEvent:
		return &WaitForEventTask{BaseTask: baseTask}, nil
	case task.TaskTypePayment:
		return &PaymentTask{BaseTask: baseTask}, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
