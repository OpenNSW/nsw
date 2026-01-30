package task

import (
	"context"
	"encoding/json"
)

type PaymentTask struct {
	CommandSet interface{}
}

func (t *PaymentTask) Start(ctx context.Context, config json.RawMessage, is StateManager, gs StateManager) (*TaskPluginReturnValue, error) {
	// Handle payment processing
	// For now, let's say it's suspended waiting for payment gateway callback
	return &TaskPluginReturnValue{
		Status:                 TaskStatusSuspended,
		StatusHumanReadableStr: "Awaiting payment",
	}, nil
}

func (t *PaymentTask) Resume(ctx context.Context, is StateManager, gs StateManager, data map[string]interface{}) (*TaskPluginReturnValue, error) {
	// Resume is called when payment is confirmed
	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: "Payment completed",
	}, nil
}
