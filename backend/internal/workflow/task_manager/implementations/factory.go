package implementations

import (
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/task_manager"
)

// taskFactory implements TaskFactory interface
type taskFactory struct{}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory() task_manager.TaskFactory {
	return &taskFactory{}
}

func (f *taskFactory) CreateTask(taskType task_manager.TaskType, taskModel *model.Task) (task_manager.Task, error) {
	baseTask := BaseTask{
		ID:        taskModel.ID,
		TaskType:  taskType,
		TaskModel: taskModel,
	}

	switch taskType {
	case task_manager.TaskTypeTraderForm:
		return &TraderFormTask{BaseTask: baseTask}, nil
	case task_manager.TaskTypeOGAForm:
		return &OGAFormTask{BaseTask: baseTask}, nil
	case task_manager.TaskTypeWaitForEvent:
		return &WaitForEventTask{BaseTask: baseTask}, nil
	case task_manager.TaskTypeDocumentSubmission:
		return &DocumentSubmissionTask{BaseTask: baseTask}, nil
	case task_manager.TaskTypePayment:
		return &PaymentTask{BaseTask: baseTask}, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
