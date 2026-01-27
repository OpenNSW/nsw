package task

import (
	"fmt"
)

// TaskFactory creates task instances from task type and model
type TaskFactory interface {
	BuildExecutor(taskType Type, commandSet interface{}, globalCtx map[string]interface{}) (ExecutionUnit, error)
}

// taskFactory implements TaskFactory interface
type taskFactory struct{}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory() TaskFactory {
	return &taskFactory{}
}

func (f *taskFactory) BuildExecutor(taskType Type, commandSet interface{}, globalCtx map[string]interface{}) (ExecutionUnit, error) {

	switch taskType {
	case TaskTypeSimpleForm:
		return NewSimpleFormTask(commandSet, globalCtx)
	case TaskTypeWaitForEvent:
		return NewWaitForEventTask(commandSet), nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
