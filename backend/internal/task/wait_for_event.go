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

func NewWaitForEventTask(commandSet interface{}, globalCtx map[string]interface{}) *WaitForEventTask {
	var config WaitForEventConfig

	// Parse the command set configuration
	if commandSet != nil {
		configBytes, err := json.Marshal(commandSet)
		if err != nil {
			slog.Error("failed to marshal command set", "error", err)
		} else {
			if err := json.Unmarshal(configBytes, &config); err != nil {
				slog.Error("failed to unmarshal wait for event config", "error", err)
			}
		}
	}

	return &WaitForEventTask{
		CommandSet: commandSet,
		globalCtx:  globalCtx,
		config:     &config,
	}
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
	go t.notifyExternalService(ctx, taskID, consignmentID)

	// Return IN_PROGRESS status immediately (non-blocking)
	// Task will be completed when external service calls back with action="complete"
	return &ExecutionResult{
		Status:  model.TaskStatusInProgress,
		Message: "Notified external service, waiting for callback",
	}, nil
}

// notifyExternalService sends task information to the configured external service
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID, consignmentID uuid.UUID) {
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

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.config.ExternalServiceURL, bytes.NewBuffer(requestBody))
	if err != nil {
		slog.ErrorContext(ctx, "failed to create HTTP request",
			"taskId", taskID,
			"consignmentId", consignmentID,
			"url", t.config.ExternalServiceURL,
			"error", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		slog.ErrorContext(ctx, "failed to send request to external service",
			"taskId", taskID,
			"consignmentId", consignmentID,
			"url", t.config.ExternalServiceURL,
			"error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.InfoContext(ctx, "successfully notified external service",
			"taskId", taskID,
			"consignmentId", consignmentID,
			"url", t.config.ExternalServiceURL,
			"status", resp.StatusCode)
	} else {
		slog.WarnContext(ctx, "external service returned non-success status",
			"taskId", taskID,
			"consignmentId", consignmentID,
			"url", t.config.ExternalServiceURL,
			"status", resp.StatusCode)
	}
}
