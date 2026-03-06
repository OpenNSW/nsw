package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
)

// Executor bundles a Plugin with its corresponding FSM.
type Executor struct {
	Plugin Plugin
	FSM    *PluginFSM
}

// TaskFactory creates task instances from the task type and model
type TaskFactory interface {
	BuildExecutor(ctx context.Context, taskType Type, config json.RawMessage) (Executor, error)
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

func (f *taskFactory) BuildExecutor(ctx context.Context, taskType Type, config json.RawMessage) (Executor, error) {
	switch taskType {
	case TaskTypeSimpleForm:
		p, err := NewSimpleForm(config, f.config, f.formService)
		return Executor{Plugin: p, FSM: NewSimpleFormFSM()}, err
	case TaskTypeWaitForEvent:
		p, err := NewWaitForEventTask(config)
		return Executor{Plugin: p, FSM: NewWaitForEventFSM()}, err
	case TaskTypePayment:
		p, err := NewPaymentTask(config)
		return Executor{Plugin: p, FSM: NewPaymentFSM()}, err
	default:
		return Executor{}, fmt.Errorf("unknown task type: %s", taskType)
	}
}
