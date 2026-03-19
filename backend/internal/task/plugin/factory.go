package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/task/plugin/gateway"
	"github.com/OpenNSW/nsw/internal/task/plugin/payment_types"
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
	repo        payment_types.PaymentRepository
	gateways    *gateway.Registry
}

// NewTaskFactory creates a new TaskFactory instance
func NewTaskFactory(cfg *config.Config, formService form.FormService, repo payment_types.PaymentRepository, gateways *gateway.Registry) TaskFactory {
	return &taskFactory{
		config:      cfg,
		formService: formService,
		repo:        repo,
		gateways:    gateways,
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
		// Determine gateway: prefer task-level config, fall back to global defaults.
		var partial struct {
			GatewayID string `json:"gatewayId"`
		}
		_ = json.Unmarshal(config, &partial) // best-effort; zero value is fine

		gwID := partial.GatewayID
		if gwID == "" {
			gwID = "govpay"
			if f.config.Payment.MockMode {
				gwID = "mock"
			}
		}
		gw, err := f.gateways.Get(gwID)
		if err != nil {
			return Executor{}, fmt.Errorf("payment gateway %q not found in registry: %w", gwID, err)
		}
		p, err := NewPaymentTask(config, f.config, f.repo, gw)
		return Executor{Plugin: p, FSM: NewPaymentFSM()}, err
	default:
		return Executor{}, fmt.Errorf("unknown task type: %s", taskType)
	}
}
