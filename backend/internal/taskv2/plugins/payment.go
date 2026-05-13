package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/pkg/remote"
)

// PaymentPlugin replaces nsw-task-flow's generic_payment. Same behaviour —
// transitions to PENDING_PAYMENT and notifies the configured payment service —
// but emits the rich body envelope so the payment service can call back to
// /api/v1/tasks/{taskId} with PAYMENT_SUCCESS / PAYMENT_FAILED.
type PaymentPlugin struct {
	client *dispatchHelper
}

func NewPaymentPlugin(manager *remote.Manager, backendBaseURL string, devMode bool) *PaymentPlugin {
	return &PaymentPlugin{client: newDispatchHelper(manager, backendBaseURL, devMode)}
}

func (p *PaymentPlugin) Name() string { return "generic_payment" }

type paymentConfig struct {
	ServiceID string `json:"service_id,omitempty"`
	Path      string `json:"path,omitempty"`
	TaskCode  string `json:"task_code,omitempty"`
}

func (p *PaymentPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg paymentConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("payment: invalid config: %w", err)
	}

	ctx.Record.Status = "PENDING_PAYMENT"

	// service_id is optional. When empty, the plugin just transitions to
	// PENDING_PAYMENT and waits for the trader's PAYMENT_SUCCESS /
	// PAYMENT_FAILED callback — there is no officer-side review for payment.
	if cfg.ServiceID == "" {
		slog.Info("taskv2 payment: no service_id configured, skipping dispatch",
			"taskId", ctx.Record.TaskID)
		return ErrSuspended
	}

	body := buildSubmissionBody(ctx.Record, &cfg.TaskCode, p.client.callbackTasksURL())

	slog.Info("taskv2 payment: dispatching to payment service",
		"taskId", ctx.Record.TaskID, "serviceId", cfg.ServiceID, "path", cfg.Path)

	if err := p.client.post(ctx.Context, cfg.ServiceID, cfg.Path, body); err != nil {
		return err
	}
	return ErrSuspended
}
