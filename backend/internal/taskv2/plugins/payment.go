package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// PaymentPlugin replaces nsw-task-flow's generic_payment. Same behaviour —
// transitions to PENDING_PAYMENT and notifies the configured payment service —
// but emits the rich body envelope so the payment service can call back to
// /api/v1/tasks/{taskId} with PAYMENT_SUCCESS / PAYMENT_FAILED.
type PaymentPlugin struct {
	client *dispatchClient
}

func NewPaymentPlugin(backendBaseURL string, devMode bool) *PaymentPlugin {
	return &PaymentPlugin{client: newDispatchClient(backendBaseURL, devMode)}
}

func (p *PaymentPlugin) Name() string { return "generic_payment" }

type paymentConfig struct {
	PaymentServiceURL string `json:"payment_service_url"`
}

func (p *PaymentPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg paymentConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("payment: invalid config: %w", err)
	}

	ctx.Record.Status = "PENDING_PAYMENT"

	// payment_service_url is optional. When empty, the plugin just transitions
	// to PENDING_PAYMENT and waits for the trader's PAYMENT_SUCCESS /
	// PAYMENT_FAILED callback — there is no officer-side review for payment.
	if cfg.PaymentServiceURL == "" {
		slog.Info("taskv2 payment: no payment_service_url configured, skipping dispatch",
			"taskId", ctx.Record.TaskID)
		return ErrSuspended
	}

	body := buildSubmissionBody(ctx.Record, nil, p.client.callbackTasksURL())

	slog.Info("taskv2 payment: dispatching to payment service",
		"taskId", ctx.Record.TaskID, "url", cfg.PaymentServiceURL)

	if err := p.client.post(ctx.Context, cfg.PaymentServiceURL, body); err != nil {
		return err
	}
	return ErrSuspended
}
