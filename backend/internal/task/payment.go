package task
 
import (
	"context"
)
 
type PaymentTask struct {
	CommandSet interface{}
}
 
func (t *PaymentTask) Start(ctx context.Context, config map[string]any) (*TaskPluginReturnValue, error) {
	// Handle payment processing
	// For now, let's say it's suspended waiting for payment gateway callback
	t.CommandSet = config
	return &TaskPluginReturnValue{
		Status:                 TaskStatusAwaitingInput,
		StatusHumanReadableStr: string(TaskStatusAwaitingInput),
		Data:                   nil,
	}, nil
}
 
func (t *PaymentTask) Resume(ctx context.Context, data map[string]any) (*TaskPluginReturnValue, error) {
	// Resume is called when payment is confirmed
	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: string(TaskStatusCompleted),
		Data:                   map[string]string{"result": "paid"},
	}, nil
}
