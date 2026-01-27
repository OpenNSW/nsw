package task

import (
	"fmt"

	"github.com/OpenNSW/nsw/internal/config"
)

// TaskFactory creates task instances from task type and model
type TaskFactory interface {
	BuildExecutor(taskType Type, commandSet interface{}, globalCtx map[string]interface{}) (ExecutionUnit, error)
}

// taskFactory implements TaskFactory interface
type taskFactory struct {
	config *config.Config
}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory(cfg *config.Config) TaskFactory {
	return &taskFactory{config: cfg}
}

func (f *taskFactory) BuildExecutor(taskType Type, commandSet interface{}, globalCtx map[string]interface{}) (ExecutionUnit, error) {

	switch taskType {
	case TaskTypeSimpleForm:
		return NewSimpleFormTask(commandSet, globalCtx, f.config)
	case TaskTypeWaitForEvent:
		return NewWaitForEventTask(commandSet), nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
