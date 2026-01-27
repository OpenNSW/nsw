package oga

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// BackendClient provides HTTP-based communication with the backend Task Manager
type BackendClient interface {
	// GetTasks fetches tasks from backend with optional filters
	GetTasks(ctx context.Context, taskType, status string) ([]BackendTask, error)
	// ExecuteTask calls POST /api/tasks/{taskId} to execute/complete a task
	ExecuteTask(ctx context.Context, taskID uuid.UUID, payload *ExecutionPayload) error
	// BaseURL returns the base URL of the backend service
	BaseURL() string
}

// BackendTask represents a task from the backend API
type BackendTask struct {
	ID            uuid.UUID       `json:"id"`
	ConsignmentID uuid.UUID       `json:"consignmentId"`
	StepID        string           `json:"stepId"`
	Type          string           `json:"type"`
	Status        string           `json:"status"`
	Config        json.RawMessage  `json:"config"`
}

// ExecutionPayload represents the payload for executing a task
type ExecutionPayload struct {
	Action  string      `json:"action"`
	Content interface{} `json:"content,omitempty"`
}

type httpBackendClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBackendClient creates a new backend client with the provided base URL.
// If baseURL is empty, it defaults to "http://localhost:8080".
func NewBackendClient(baseURL string) BackendClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080" // Default: backend service port
	}

	return &httpBackendClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *httpBackendClient) BaseURL() string {
	return c.baseURL
}

func (c *httpBackendClient) GetTasks(ctx context.Context, taskType, status string) ([]BackendTask, error) {
	url := fmt.Sprintf("%s/api/tasks", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	if taskType != "" {
		q.Add("type", taskType)
	}
	if status != "" {
		q.Add("status", status)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var tasks []BackendTask
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tasks, nil
}

func (c *httpBackendClient) ExecuteTask(ctx context.Context, taskID uuid.UUID, payload *ExecutionPayload) error {
	url := fmt.Sprintf("%s/api/tasks/%s", c.baseURL, taskID)
	
	// Request body only needs payload, taskId is in URL path
	reqBody := map[string]interface{}{
		"payload": payload,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.WarnContext(ctx, "backend task execution returned non-OK status",
			"taskID", taskID,
			"statusCode", resp.StatusCode,
			"body", string(body))
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
