package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

// WaitForEventConfig represents the configuration for a WAIT_FOR_EVENT task
type WaitForEventConfig struct {
	ExternalServiceURL string `json:"externalServiceUrl"` // URL of the external service to notify
}

type WaitForEventTask struct {
}

// Start initializes the task, sends notification to external service, and suspends
func (t *WaitForEventTask) Start(ctx context.Context, config map[string]any) (*TaskPluginReturnValue, error) {
	// Parse config
	var configStruct WaitForEventConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &configStruct,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode task config: %w", err)
	}
	
	if configStruct.ExternalServiceURL == "" {
		return nil, fmt.Errorf("externalServiceUrl not configured in task config")
	}

	// Extract IDs from config (added by TaskContainer/Manager)
	taskIDVal, ok := config["taskId"]
	if !ok {
		return nil, fmt.Errorf("taskId missing from config")
	}
	taskIDStr, ok := taskIDVal.(string)
	if !ok {
		return nil, fmt.Errorf("taskId is not a string")
	}

	consignmentIDVal, ok := config["consignmentId"]
	if !ok {
		return nil, fmt.Errorf("consignmentId missing from config")
	}
	consignmentIDStr, ok := consignmentIDVal.(string)
	if !ok {
		return nil, fmt.Errorf("consignmentId is not a string")
	}

	if taskIDStr == "" || consignmentIDStr == "" {
		return nil, fmt.Errorf("taskId or consignmentId is empty")
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid taskId format: %w", err)
	}
	consignmentID, err := uuid.Parse(consignmentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid consignmentId format: %w", err)
	}

	// Notify external service asynchronously
	// We use a detached context or background context to ensure it runs even if Start returns quickly
	notifyCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	go func() {
		defer cancel()
		t.notifyExternalService(notifyCtx, taskID, consignmentID, configStruct.ExternalServiceURL)
	}()

	return &TaskPluginReturnValue{
		Status:                 TaskStatusAwaitingInput,
		StatusHumanReadableStr: string(TaskStatusAwaitingInput),
		Data:                   nil,
	}, nil
}

func (t *WaitForEventTask) Resume(ctx context.Context, data map[string]any) (*TaskPluginReturnValue, error) {
	// Resume is called when the event occurs
	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: string(TaskStatusCompleted),
		Data:                   map[string]string{"message": "Event received"},
	}, nil
}

// notifyExternalService sends task information to the configured external service with retry logic
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID, consignmentID uuid.UUID, serviceURL string) {
	const (
		maxRetries     = 3
		initialBackoff = 1 * time.Second
	)

	// ExternalServiceRequest represents the payload sent to the external service
	type ExternalServiceRequest struct {
		ConsignmentID uuid.UUID `json:"consignmentId"`
		TaskID        uuid.UUID `json:"taskId"`
	}

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
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceURL, bytes.NewBuffer(requestBody))
		if err != nil {
			slog.ErrorContext(ctx, "failed to create HTTP request",
				"taskId", taskID,
				"consignmentId", consignmentID,
				"url", serviceURL,
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
				"url", serviceURL,
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
				"url", serviceURL,
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
				"url", serviceURL,
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
				"url", serviceURL,
				"status", resp.StatusCode)
			break
		}
	}

	// All retries exhausted or non-retryable error occurred
	slog.ErrorContext(ctx, "failed to notify external service after all retries - task may be stuck in IN_PROGRESS",
		"taskId", taskID,
		"consignmentId", consignmentID,
		"url", serviceURL,
		"maxRetries", maxRetries,
		"error", lastErr)
}
