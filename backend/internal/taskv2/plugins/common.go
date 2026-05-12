// Package plugins provides drop-in replacements for the nsw-task-flow
// dispatching plugins (generic_external_review, register_task_and_wait,
// generic_payment, generic_http_post).
//
// They keep the same plugin Name()s — so registration and registry routing
// are unchanged — but they:
//
//  1. Read the in-memory TaskRecord pointer directly. nsw-task-flow's stock
//     plugins delegate to a callback dispatcher that has no access to the
//     record, which means a Store.GetTask() lookup from inside the dispatcher
//     returns stale data (TaskManager.StartSubTask saves the record AFTER the
//     plugin runs, not before).
//
//  2. POST a richer body shape that matches the OpenNSW OGA SimpleForm
//     contract (taskCode, taskId, workflowId, serviceUrl, data, …) so the
//     external NPQS / FCAU portals at /api/oga/inject can consume it
//     unchanged.
//
//  3. Honour devMode — if dispatch fails (e.g. the OGA portal isn't running
//     yet) the plugin still transitions the task to its waiting state and
//     logs a warning, so local development doesn't block the workflow.
package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	tfplugins "github.com/OpenNSW/nsw-task-flow/plugins"
)

// dispatchClient bundles outbound HTTP behaviour shared by every dispatching
// plugin in this package.
type dispatchClient struct {
	httpClient     *http.Client
	backendBaseURL string
	devMode        bool
}

func newDispatchClient(backendBaseURL string, devMode bool) *dispatchClient {
	return &dispatchClient{
		httpClient:     &http.Client{Timeout: 15 * time.Second},
		backendBaseURL: strings.TrimRight(backendBaseURL, "/"),
		devMode:        devMode,
	}
}

// callbackTasksURL is the URL the receiving OGA portal should call back into
// to advance the workflow once the officer has acted.
func (c *dispatchClient) callbackTasksURL() string {
	return c.backendBaseURL + "/api/v1/tasks"
}

// post sends body as JSON to url and returns nil on any 2xx. In devMode, dispatch
// errors are logged-and-swallowed so the workflow can still be driven via the
// in-process OGA-app (POST /api/oga/applications/{id}/review).
func (c *dispatchClient) post(ctx context.Context, url string, body any) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.dispatchOrSwallow("transport", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("taskv2 plugin: failed to close response body", "url", url, "error", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		err := fmt.Errorf("POST %s returned %d: %s", url, resp.StatusCode, preview)
		return c.dispatchOrSwallow("status", url, err)
	}
	return nil
}

func (c *dispatchClient) dispatchOrSwallow(kind, url string, err error) error {
	if c.devMode {
		slog.Warn("taskv2 plugin: dispatch failed (dev mode — swallowing)",
			"kind", kind, "url", url, "error", err)
		return nil
	}
	return err
}

// pluginContext is just an alias so plugin signatures stay tidy.
type pluginContext = tfplugins.PluginContext

// ErrSuspended signals to the orchestrator that this plugin step is parked and
// waiting for an external callback before the sub-workflow can advance.
var ErrSuspended = tfplugins.ErrSuspended
