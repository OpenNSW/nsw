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

// WaitForEventConfig represents the configuration for a WAIT_FOR_EVENT task
type WaitForEventConfig struct {
	ExternalServiceURL string // URL of the external service to notify
}

type WaitForEventTask struct {
	api    API
	config WaitForEventConfig
}

func (t *WaitForEventTask) Init(api API) {
	t.api = api
}

func (t *WaitForEventTask) Start(ctx context.Context) (*ExecutionResponse, error) {
	return &ExecutionResponse{
		Message: "WAIT_FOR_EVENT task started successfully.",
	}, nil
}

// ExternalServiceRequest represents the payload sent to the external service
type ExternalServiceRequest struct {
	ConsignmentID *uuid.UUID `json:"consignmentId,omitempty"`
	TaskID        uuid.UUID  `json:"taskId"`
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
	// Handle completion action from external service callback
	if request != nil && request.Action == "complete" {
		completedState := Completed
		return &ExecutionResponse{
			NewState: &completedState,
			Message:  "Task completed by external service",
		}, nil
	}

	// Extract task and consignment IDs from global context
	taskID := t.api.GetTaskID()
	consignmentID := t.api.GetConsignmentID()

	// Validate external service URL
	if t.config.ExternalServiceURL == "" {
		return nil, fmt.Errorf("externalServiceUrl not configured in task config")
	}

	// Send task information to external service asynchronously
	// Use background context with timeout to ensure notification completes
	// independently of the Execute method's context lifecycle
	notifyCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	go func() {
		defer cancel()
		t.notifyExternalService(notifyCtx, taskID, consignmentID)
	}()

	// Return IN_PROGRESS status immediately (non-blocking)
	// Task will be completed when external service calls back with action="complete"
	inProgressState := InProgress
	return &ExecutionResponse{
		NewState: &inProgressState,
		Message:  "Notified external service, waiting for callback",
	}, nil
}

// notifyExternalService sends task information to the configured external service with retry logic
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID uuid.UUID, consignmentID *uuid.UUID) {
	const (
		maxRetries     = 3
		initialBackoff = 1 * time.Second
	)

	request := ExternalServiceRequest{
		ConsignmentID: consignmentID,
		TaskID:        taskID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal external service request",
			"taskId", taskID,
			"consignmentId", consignmentID,
			"error", err)
		return
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
				"consignmentId", consignmentID,
				"attempt", attempt+1)
			return
		default:
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.config.ExternalServiceURL, bytes.NewBuffer(requestBody))
		if err != nil {
			slog.ErrorContext(ctx, "failed to create HTTP request",
				"taskId", taskID,
				"consignmentId", consignmentID,
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
				"consignmentId", consignmentID,
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
						"consignmentId", consignmentID)
					return
				}
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.InfoContext(ctx, "successfully notified external service",
				"taskId", taskID,
				"consignmentId", consignmentID,
				"url", t.config.ExternalServiceURL,
				"status", resp.StatusCode,
				"attempt", attempt+1)
			return
		}

		// Retry on server errors (5xx) and rate limit (429)
		if (resp.StatusCode >= 500 && resp.StatusCode < 600) || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("external service returned status %d", resp.StatusCode)
			slog.WarnContext(ctx, "external service returned retryable error status",
				"taskId", taskID,
				"consignmentId", consignmentID,
				"url", t.config.ExternalServiceURL,
				"status", resp.StatusCode,
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
						"consignmentId", consignmentID)
					return
				}
			}
		} else {
			// Non-retryable client error (4xx other than 429)
			lastErr = fmt.Errorf("external service returned non-retryable status %d", resp.StatusCode)
			slog.ErrorContext(ctx, "external service returned non-retryable error status",
				"taskId", taskID,
				"consignmentId", consignmentID,
				"url", t.config.ExternalServiceURL,
				"status", resp.StatusCode)
			break
		}
	}

	// All retries exhausted or non-retryable error occurred
	slog.ErrorContext(ctx, "failed to notify external service after all retries - task may be stuck in IN_PROGRESS",
		"taskId", taskID,
		"consignmentId", consignmentID,
		"url", t.config.ExternalServiceURL,
		"maxRetries", maxRetries,
		"error", lastErr)
}
