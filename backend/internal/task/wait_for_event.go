package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// WaitForEventConfig represents the configuration for a WAIT_FOR_EVENT task
type WaitForEventConfig struct {
	ExternalServiceURL string `json:"externalServiceUrl"` // URL of the external service to notify
}

type WaitForEventTask struct {
	CommandSet interface{}
	globalCtx  map[string]interface{}
	config     *WaitForEventConfig
}

// ExternalServiceRequest represents the payload sent to the external service
type ExternalServiceRequest struct {
	ConsignmentID uuid.UUID `json:"consignmentId"`
	TaskID        uuid.UUID `json:"taskId"`
}

func NewWaitForEventTask(commandSet interface{}, globalCtx map[string]interface{}) (*WaitForEventTask, error) {
	var config WaitForEventConfig

	// Parse the command set configuration
	if commandSet != nil {
		configBytes, err := json.Marshal(commandSet)
		if err != nil {
			slog.Error("failed to marshal command set", "error", err)
			return nil, fmt.Errorf("failed to marshal command set: %w", err)
		} else {
			if err := json.Unmarshal(configBytes, &config); err != nil {
				slog.Error("failed to unmarshal wait for event config", "error", err)
				return nil, fmt.Errorf("failed to unmarshal wait for event config: %w", err)
			}
		}
	}

	return &WaitForEventTask{
		CommandSet: commandSet,
		globalCtx:  globalCtx,
		config:     &config,
	}, nil
}

func (t *WaitForEventTask) Execute(ctx context.Context, payload *ExecutionPayload) (*ExecutionResult, error) {
	// Handle completion action from external service callback
	if payload != nil && payload.Action == "complete" {
		slog.InfoContext(ctx, "task completion received from external service",
			"taskId", t.globalCtx["taskId"],
			"consignmentId", t.globalCtx["consignmentId"])

		return &ExecutionResult{
			Status:  model.TaskStatusCompleted,
			Message: "Task completed by external service",
		}, nil
	}

	// Extract task and consignment IDs from global context
	taskID, ok := t.globalCtx["taskId"].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("taskId not found in global context")
	}

	consignmentID, ok := t.globalCtx["consignmentId"].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("consignmentId not found in global context")
	}

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
	return &ExecutionResult{
		Status:  model.TaskStatusInProgress,
		Message: "Notified external service, waiting for callback",
	}, nil
}

// notifyExternalService sends task information to the configured external service with retry logic
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID, consignmentID uuid.UUID) {
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
