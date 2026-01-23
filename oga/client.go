package oga

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

// Client provides HTTP-based communication with OGA service
type Client interface {
	// NotifyApplicationReady notifies OGA service that an application is ready for review
	NotifyApplicationReady(ctx context.Context, notification OGATaskNotification) error
	// NotifyTaskCompleted notifies OGA service that a task has been completed or rejected
	NotifyTaskCompleted(ctx context.Context, taskID uuid.UUID) error
	// BaseURL returns the base URL of the OGA service
	BaseURL() string
}

type httpClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new OGA client with the provided base URL.
// If baseURL is empty, it defaults to "http://localhost:8081".
// For different ports or hosts, set the OGA_SERVICE_URL environment variable
// or pass a non-empty baseURL parameter.
func NewClient(baseURL string) Client {
	if baseURL == "" {
		baseURL = "http://localhost:8081" // Default: OGA service port (can be overridden via OGA_SERVICE_URL env var)
	}

	return &httpClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *httpClient) NotifyApplicationReady(ctx context.Context, notification OGATaskNotification) error {
	url := fmt.Sprintf("%s/api/oga/notifications", c.baseURL)
	return c.sendNotification(ctx, url, notification)
}

func (c *httpClient) NotifyTaskCompleted(ctx context.Context, taskID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/oga/tasks/%s/completed", c.baseURL, taskID)
	return c.sendNotification(ctx, url, nil)
}

func (c *httpClient) sendNotification(ctx context.Context, url string, body interface{}) error {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal notification: %w", err)
		}
	}

	maxRetries := 3
	backoff := 1 * time.Second

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		var req *http.Request
		if body != nil {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
		} else {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		}
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 400 {
				return nil
			}
			lastErr = fmt.Errorf("oga service returned status: %s", resp.Status)
		} else {
			lastErr = err
		}

		slog.WarnContext(ctx, "failed to notify OGA service, retrying...",
			"attempt", i+1,
			"url", url,
			"error", lastErr,
			"backoff", backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return fmt.Errorf("failed to notify OGA service at %s after %d attempts: %w", url, maxRetries, lastErr)
}

func (c *httpClient) BaseURL() string {
	return c.baseURL
}

// NoopClient is a no-op implementation of Client for testing or when OGA is disabled
type NoopClient struct{}

func (c *NoopClient) NotifyApplicationReady(_ context.Context, _ OGATaskNotification) error {
	return nil
}

func (c *NoopClient) NotifyTaskCompleted(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (c *NoopClient) BaseURL() string {
	return ""
}

// NewNoopClient creates a no-op OGA client
func NewNoopClient() Client {
	return &NoopClient{}
}
