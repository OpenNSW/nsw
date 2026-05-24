package plugins

import (
	"context"
	"log/slog"
	"net/url"

	tfplugins "github.com/OpenNSW/nsw-task-flow/plugins"
	"github.com/OpenNSW/nsw/pkg/remote"
)

// Package plugins provide drop-in replacements for the nsw-task-flow
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
//
//  4. Resolve target URLs via remote.Manager so service base URLs,
//     authentication, and timeouts are configured centrally in services.json.
//     Template configs specify only service_id + path, never full URLs.

// dispatchHelper bundles outbound HTTP behaviour shared by every dispatching
// plugin in this package. It delegates to remote.Manager so service base URLs,
// authentication, and timeouts are configured centrally in services.json
// rather than being hard-coded in the template configs.
type dispatchHelper struct {
	manager        *remote.Manager
	backendBaseURL string
	devMode        bool
}

func newDispatchHelper(manager *remote.Manager, backendBaseURL string, devMode bool) *dispatchHelper {
	return &dispatchHelper{
		manager:        manager,
		backendBaseURL: backendBaseURL,
		devMode:        devMode,
	}
}

// callbackTasksURL is the URL the receiving OGA portal should call back into
// to advance the workflow once the officer has acted.
func (h *dispatchHelper) callbackTasksURL() string {
	joined, err := url.JoinPath(h.backendBaseURL, "/api/v1/tasks")
	if err != nil {
		slog.Error("taskv2 plugin: failed to build callback URL",
			"backendBaseURL", h.backendBaseURL, "error", err)
		return h.backendBaseURL + "/api/v1/tasks"
	}
	return joined
}

// post sends body as JSON to the resolved service+path and returns nil on any
// 2xx. In devMode, dispatch errors are logged-and-swallowed so the workflow
// can still be driven via the in-process OGA-app.
func (h *dispatchHelper) post(ctx context.Context, serviceID, path string, body any) error {
	req := remote.Request{
		Method: "POST",
		Path:   path,
		Body:   body,
	}
	if err := h.manager.Call(ctx, serviceID, req, nil); err != nil {
		return h.dispatchOrSwallow(serviceID, path, err)
	}
	return nil
}

func (h *dispatchHelper) dispatchOrSwallow(serviceID, path string, err error) error {
	if h.devMode {
		slog.Warn("taskv2 plugin: dispatch failed (dev mode — swallowing)",
			"serviceId", serviceID, "path", path, "error", err)
		return nil
	}
	return err
}

// pluginContext is just an alias so plugin signatures stay tidy.
type pluginContext = tfplugins.PluginContext

// ErrSuspended signals to the orchestrator that this plugin step is parked and
// waiting for an external callback before the sub-workflow can advance.
var ErrSuspended = tfplugins.ErrSuspended
