package task

import (
	"fmt"
)

// TaskFactory creates task instances from task type and model
type TaskFactory interface {
	BuildExecutor(taskType Type, commandSet interface{}) (ExecutionUnit, error)
}

// taskFactory implements TaskFactory interface
type taskFactory struct{}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory() TaskFactory {
	return &taskFactory{}
}

func (f *taskFactory) BuildExecutor(taskType Type, commandSet interface{}) (ExecutionUnit, error) {

	switch taskType {
	case TaskTypeTraderForm:
		return NewTraderFormTask(commandSet)
	case TaskTypeOGAForm:
		return &OGAFormTask{CommandSet: commandSet}, nil
	case TaskTypeWaitForEvent:
		return &WaitForEventTask{CommandSet: commandSet}, nil
	case TaskTypePayment:
		return &PaymentTask{CommandSet: commandSet}, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
