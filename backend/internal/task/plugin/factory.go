package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/pkg/remote"
	"gorm.io/gorm"
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
	config        *config.Config
	formService   form.FormService
	remoteManager *remote.Manager
}

// NewTaskFactory creates a new TaskFactory instance and initializes the remote services manager.
func NewTaskFactory(cfg *config.Config, db *gorm.DB) TaskFactory {
	rm := remote.NewManager()
	if err := rm.LoadServices(cfg.Server.ServicesConfigPath); err != nil {
		slog.Warn("factory: failed to load external services configuration",
			"path", cfg.Server.ServicesConfigPath,
			"error", err)
	} else {
		slog.Info("factory: external services configuration loaded",
			"services", rm.ListServices())
	}

	return &taskFactory{
		config:        cfg,
		remoteManager: rm,
		formService:   form.NewFormService(db),
	}
}

func (f *taskFactory) BuildExecutor(ctx context.Context, taskType Type, config json.RawMessage) (Executor, error) {
	switch taskType {
	case TaskTypeSimpleForm:
		p, err := NewSimpleForm(config, f.config, f.formService, f.remoteManager)
		return Executor{Plugin: p, FSM: NewSimpleFormFSM()}, err
	case TaskTypeWaitForEvent:
		p, err := NewWaitForEventTask(config, f.remoteManager)
		return Executor{Plugin: p, FSM: NewWaitForEventFSM()}, err
	case TaskTypePayment:
		p, err := NewPaymentTask(config)
		return Executor{Plugin: p, FSM: NewPaymentFSM()}, err
	default:
		return Executor{}, fmt.Errorf("unknown task type: %s", taskType)
	}
}
