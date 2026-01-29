package task

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
)

// TaskFactory creates task instances from task type and model
	// BuildPlugin creates a TaskPlugin for the given task type
	BuildPlugin(taskType string) (TaskPlugin, error)
}

// taskFactory implements TaskFactory interface
type taskFactory struct {
	config      *config.Config
	formService form.FormService
}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory(cfg *config.Config, formService form.FormService) TaskFactory {
	return &taskFactory{
		config:      cfg,
		formService: formService,
	}
}

func (f *taskFactory) BuildPlugin(taskType string) (TaskPlugin, error) {

	switch taskType {
	case "SIMPLE_FORM":
		return NewSimpleFormTask(f.config, f.formService)
	case "WAIT_FOR_EVENT":
		return &WaitForEventTask{}, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

