package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type waitForEventState string

const (
	notifiedService  waitForEventState = "NOTIFIED_SERVICE"
	receivedCallback waitForEventState = "RECEIVED_CALLBACK"
)

// WaitForEventConfig represents the configuration for a WAIT_FOR_EVENT task
type WaitForEventConfig struct {
	ExternalServiceURL string // URL of the external service to notify
}

type WaitForEventTask struct {
	api    API
	config WaitForEventConfig
}

func (t *WaitForEventTask) GetRenderInfo(_ context.Context) (*ApiResponse, error) {
	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypeWaitForEvent,
			PluginState: t.api.GetPluginState(),
			State:       t.api.GetTaskState(),
			Content:     nil, // No specific content needed for rendering
		},
	}, nil
}

func (t *WaitForEventTask) Init(api API) {
	t.api = api
}

func (t *WaitForEventTask) Start(ctx context.Context) (*ExecutionResponse, error) {

	// Extract task and workflow IDs from global context
	taskID := t.api.GetTaskID()
	workflowID := t.api.GetWorkflowID()

	// Validate external service URL
	if t.config.ExternalServiceURL == "" {
		return nil, fmt.Errorf("externalServiceUrl not configured in task config")
	}

	// Notify external service synchronously — only transition to InProgress on success
	if err := t.notifyExternalService(ctx, taskID, workflowID); err != nil {
		return nil, fmt.Errorf("failed to notify external service: %w", err)
	}

	pluginState := string(notifiedService)
	if err := t.api.SetPluginState(pluginState); err != nil {
		slog.ErrorContext(ctx, "failed to set plugin state after notifying external service",
			"taskId", taskID,
			"workflowId", workflowID,
			"error", err)
		return nil, fmt.Errorf("failed to set plugin state after notifying external service: %w", err)
	}

	// Task will be completed when external service calls back with action="complete"
	inProgressState := InProgress
	return &ExecutionResponse{
		ExtendedState: &pluginState,
		NewState:      &inProgressState,
		Message:       "Notified external service, waiting for callback",
	}, nil
}

// ExternalServiceRequest represents the payload sent to the external service
type ExternalServiceRequest struct {
	WorkflowID uuid.UUID `json:"workflowId"`
	TaskID     uuid.UUID `json:"taskId"`
}

func NewWaitForEventTask(raw json.RawMessage) (*WaitForEventTask, error) {
	var cfg WaitForEventConfig

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}

	return &WaitForEventTask{
		config: cfg,
	}, nil
}

func (t *WaitForEventTask) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("execution request is required")
	}

	// Handle completion action from external service callback
	if request.Action == "complete" {
		pluginState := string(receivedCallback)
		if err := t.api.SetPluginState(pluginState); err != nil {
			slog.ErrorContext(ctx, "failed to set plugin state on receiving callback",
				"taskId", t.api.GetTaskID(),
				"workflowId", t.api.GetWorkflowID(),
				"error", err)
			return nil, fmt.Errorf("failed to set plugin state on receiving callback: %w", err)
		}

		completedState := Completed
		return &ExecutionResponse{
			ExtendedState: &pluginState,
			NewState:      &completedState,
			Message:       "Task completed by external service",
		}, nil
	}

	return nil, fmt.Errorf("unsupported action: %s", request.Action)
}

// notifyExternalService sends task information to the configured external service with retry logic
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID uuid.UUID, workflowID uuid.UUID) error {
	const (
		maxRetries     = 3
		initialBackoff = 1 * time.Second
	)

	request := ExternalServiceRequest{
		WorkflowID: workflowID,
		TaskID:     taskID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal external service request",
			"taskId", taskID,
			"workflowId", workflowID,
			"error", err)
		return err
	}

	var lastErr error
	backoff := initialBackoff

	// Reuse HTTP client across retry attempts for connection pooling
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			slog.WarnContext(ctx, "context cancelled before external service notification",
				"taskId", taskID,
				"workflowId", workflowID,
				"attempt", attempt+1)
			return ctx.Err()
		default:
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.config.ExternalServiceURL, bytes.NewBuffer(requestBody))
		if err != nil {
			slog.ErrorContext(ctx, "failed to create HTTP request",
				"taskId", taskID,
				"workflowId", workflowID,
				"url", t.config.ExternalServiceURL,
				"attempt", attempt+1,
				"error", err)
			lastErr = err
			// Don't retry on request creation errors
			break
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			slog.WarnContext(ctx, "failed to send request to external service",
				"taskId", taskID,
				"workflowId", workflowID,
				"url", t.config.ExternalServiceURL,
				"attempt", attempt+1,
				"maxRetries", maxRetries,
				"error", err)

			// Retry on network errors
			if attempt < maxRetries {
				select {
				case <-time.After(backoff):
					backoff *= 2 // Exponential backoff
					continue
				case <-ctx.Done():
					slog.WarnContext(ctx, "context cancelled during external service retry",
						"taskId", taskID,
						"workflowId", workflowID)
					return ctx.Err()
				}
			}
			continue
		}
		statusCode := resp.StatusCode
		_ = resp.Body.Close()

		if statusCode >= 200 && statusCode < 300 {
			slog.InfoContext(ctx, "successfully notified external service",
				"taskId", taskID,
				"workflowId", workflowID,
				"url", t.config.ExternalServiceURL,
				"status", statusCode,
				"attempt", attempt+1)
			return nil
		}

		// Retry on server errors (5xx) and rate limit (429)
		if (statusCode >= 500 && statusCode < 600) || statusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("external service returned status %d", statusCode)
			slog.WarnContext(ctx, "external service returned retryable error status",
				"taskId", taskID,
				"workflowId", workflowID,
				"url", t.config.ExternalServiceURL,
				"status", statusCode,
				"attempt", attempt+1,
				"maxRetries", maxRetries)

			if attempt < maxRetries {
				select {
				case <-time.After(backoff):
					backoff *= 2 // Exponential backoff
					continue
				case <-ctx.Done():
					slog.WarnContext(ctx, "context cancelled during external service retry",
						"taskId", taskID,
						"workflowId", workflowID)
					return ctx.Err()
				}
			}
		} else {
			// Non-retryable client error (4xx other than 429)
			lastErr = fmt.Errorf("external service returned non-retryable status %d", statusCode)
			slog.ErrorContext(ctx, "external service returned non-retryable error status",
				"taskId", taskID,
				"workflowId", workflowID,
				"url", t.config.ExternalServiceURL,
				"status", statusCode)
			break
		}
	}

	// All retries exhausted or non-retryable error occurred
	slog.ErrorContext(ctx, "failed to notify external service after all retries",
		"taskId", taskID,
		"workflowId", workflowID,
		"url", t.config.ExternalServiceURL,
		"maxRetries", maxRetries,
		"error", lastErr)
	return lastErr
}
