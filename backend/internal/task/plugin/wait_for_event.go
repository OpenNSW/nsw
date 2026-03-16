package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/pkg/remote"
)

type waitForEventState string

const (
	notifiedService  waitForEventState = "NOTIFIED_SERVICE"
	notifyFailed     waitForEventState = "NOTIFY_FAILED"
	receivedCallback waitForEventState = "RECEIVED_CALLBACK"
)

// Internal FSM actions for WaitForEventTask.
const (
	waitForEventFSMStartFailed = "START_FAILED"
	waitForEventFSMRetry       = "RETRY"
	waitForEventFSMComplete    = "COMPLETE"
)

// WaitForEventDisplay holds optional UI display metadata for the portal
type WaitForEventDisplay struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// WaitForEventConfig represents the configuration for a WAIT_FOR_EVENT task
type WaitForEventConfig struct {
	ServiceID          string               `json:"serviceId"`
	ExternalServiceURL string               `json:"externalServiceUrl"`
	Display            *WaitForEventDisplay `json:"display,omitempty"`
}

type WaitForEventTask struct {
	api           API
	config        WaitForEventConfig
	remoteManager *remote.Manager
}

func (t *WaitForEventTask) GetRenderInfo(_ context.Context) (*ApiResponse, error) {
	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypeWaitForEvent,
			PluginState: t.api.GetPluginState(),
			State:       t.api.GetTaskState(),
			Content:     t.renderContent(),
		},
	}, nil
}

func (t *WaitForEventTask) Init(api API) {
	t.api = api
}

// ExternalServiceRequest represents the payload sent to the external service
type ExternalServiceRequest struct {
	WorkflowID string `json:"workflowId"`
	TaskID     string `json:"taskId"`
}

// NewWaitForEventFSM returns the state graph for WaitForEventTask.
//
// State graph:
//
//	""               ──START────────► NOTIFIED_SERVICE  [IN_PROGRESS]
//	""               ──START_FAILED─► NOTIFY_FAILED     [IN_PROGRESS]
//	NOTIFY_FAILED    ──RETRY────────► NOTIFIED_SERVICE  [IN_PROGRESS]
//	NOTIFIED_SERVICE ──COMPLETE─────► RECEIVED_CALLBACK [COMPLETED]
func NewWaitForEventFSM() *PluginFSM {
	return NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}:                               {string(notifiedService), InProgress},
		{"", waitForEventFSMStartFailed}:                   {string(notifyFailed), InProgress},
		{string(notifyFailed), waitForEventFSMRetry}:       {string(notifiedService), InProgress},
		{string(notifiedService), waitForEventFSMComplete}: {string(receivedCallback), Completed},
	})
}

func (t *WaitForEventTask) renderContent() map[string]any {
	content := map[string]any{}
	if t.config.Display != nil {
		content["display"] = t.config.Display
	}
	return content
}

func NewWaitForEventTask(raw json.RawMessage, remoteManager *remote.Manager) (*WaitForEventTask, error) {
	var cfg WaitForEventConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &WaitForEventTask{config: cfg, remoteManager: remoteManager}, nil
}

func (t *WaitForEventTask) Start(ctx context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(FSMActionStart) {
		return &ExecutionResponse{Message: "WaitForEvent already started"}, nil
	}
	if t.config.ExternalServiceURL == "" {
		return nil, fmt.Errorf("externalServiceUrl not configured in task config")
	}
	if err := t.notifyExternalService(ctx, t.api.GetTaskID(), t.api.GetWorkflowID()); err != nil {
		if transErr := t.api.Transition(waitForEventFSMStartFailed); transErr != nil {
			slog.ErrorContext(ctx, "failed to transition to NOTIFY_FAILED after notification error",
				"taskId", t.api.GetTaskID(),
				"workflowId", t.api.GetWorkflowID(),
				"error", transErr)
		}
		return &ExecutionResponse{Message: "Failed to notify external service"}, nil
	}
	if err := t.api.Transition(FSMActionStart); err != nil {
		return nil, err
	}
	return &ExecutionResponse{Message: "Notified external service, waiting for callback"}, nil
}

func (t *WaitForEventTask) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("execution request is required")
	}
	switch request.Action {
	case waitForEventFSMRetry:
		if err := t.notifyExternalService(ctx, t.api.GetTaskID(), t.api.GetWorkflowID()); err != nil {
			// error is nil since the problem is not on system side.
			return &ExecutionResponse{
				Message: "Failed to notify external service",
				ApiResponse: &ApiResponse{
					Success: false,
					Error: &ApiError{
						Code:    "EXTERNAL_SERVICE_NOTIFICATION_FAILED",
						Message: "Failed to notify external service",
					},
				},
			}, nil
		}
		if err := t.api.Transition(waitForEventFSMRetry); err != nil {
			return nil, err
		}
		return &ExecutionResponse{
			Message: "Notified external service, waiting for callback",
			ApiResponse: &ApiResponse{
				Success: true,
			},
		}, nil
	case waitForEventFSMComplete:
		if err := t.api.Transition(waitForEventFSMComplete); err != nil {
			return nil, err
		}
		return &ExecutionResponse{
			Message: "Task completed by external service",
			ApiResponse: &ApiResponse{
				Success: true,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported action %q for WaitForEventTask", request.Action)
	}
}

// notifyExternalService sends task information to the configured external service with retry logic
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID string, workflowID string) error {
	target := t.config.ExternalServiceURL
	if target == "" {
		return fmt.Errorf("externalServiceUrl not configured")
	}

	req := remote.Request{
		Method: "POST",
		Path:   target,
		Body: ExternalServiceRequest{
			WorkflowID: workflowID,
			TaskID:     taskID,
		},
		Retry: &remote.DefaultRetryConfig,
	}

	// 1. Try to use the Manager if it's available.
	// This will handle serviceID lookups and URL-based identification with auth.
	if t.remoteManager == nil {
		return fmt.Errorf("remote manager not initialized and target %q is not a valid URL", target)
	}
	// Manager.Call will attempt to resolve the service ID from the Path if serviceID is empty.
	if err := t.remoteManager.Call(ctx, t.config.ServiceID, req, nil); err != nil {
		return fmt.Errorf("failed to notify external service %q: %w", target, err)
	}
	return nil
}
