package plugin

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
)

// TaskFactory creates task instances from the task type and model
type TaskFactory interface {
	BuildExecutor(ctx context.Context, taskType Type, config map[string]any) (Plugin, error)
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

func (f *taskFactory) BuildExecutor(ctx context.Context, taskType Type, config map[string]any) (Plugin, error) {

	switch taskType {
	case TaskTypeSimpleForm:
		return NewSimpleForm(config, f.config, f.formService)
	case TaskTypeWaitForEvent:
		return NewWaitForEventTask(config)
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
