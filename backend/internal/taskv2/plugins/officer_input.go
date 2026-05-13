package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"

	tfplugins "github.com/OpenNSW/nsw-task-flow/plugins"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"
	"github.com/OpenNSW/nsw/pkg/remote"
)

// OfficerInputPlugin is our drop-in replacement for nsw-task-flow's
// generic_officer_input. The stock plugin only flips status to
// QUEUED_EXTERNALLY — it does not notify the OGA portal — so officer-only
// steps (e.g. issue certificate) never appear in the officer's queue.
//
// This version mirrors ExternalReviewPlugin: it sets ReviewerFormID and
// QUEUED_EXTERNALLY, then POSTs the same SimpleForm envelope the OGA
// portal at /api/oga/inject expects, so the task shows up for the officer.
//
// `service_id` and `path` are optional. When omitted, the plugin still
// transitions to QUEUED_EXTERNALLY so the in-process oga-router queue surfaces
// it; the dispatch is what's needed for the standalone NPQS / FCAU OGA portals.
type OfficerInputPlugin struct {
	client *dispatchHelper
}

func NewOfficerInputPlugin(manager *remote.Manager, backendBaseURL string, devMode bool) *OfficerInputPlugin {
	return &OfficerInputPlugin{client: newDispatchHelper(manager, backendBaseURL, devMode)}
}

func (p *OfficerInputPlugin) Name() string { return "generic_officer_input" }

type officerInputConfig struct {
	OfficerJsonFormsID string `json:"officer_jsonforms_id,omitempty"`
	ServiceID          string `json:"service_id,omitempty"`
	Path               string `json:"path,omitempty"`
	TaskCode           string `json:"task_code,omitempty"`
	StatusOverride     string `json:"status_override,omitempty"`
}

func (p *OfficerInputPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg officerInputConfig
	if len(configRaw) > 0 && string(configRaw) != "null" {
		if err := json.Unmarshal(configRaw, &cfg); err != nil {
			return fmt.Errorf("officer_input: invalid config: %w", err)
		}
	}

	status := "QUEUED_EXTERNALLY"
	if cfg.StatusOverride != "" {
		status = cfg.StatusOverride
	}

	if cfg.OfficerJsonFormsID != "" {
		ctx.Record.ReviewerFormID = cfg.OfficerJsonFormsID
	}
	ctx.Record.Status = status

	if cfg.ServiceID == "" {
		slog.Info("taskv2 officer_input: no service_id configured, skipping dispatch",
			"taskId", ctx.Record.TaskID, "form", ctx.Record.ReviewerFormID)
		return ErrSuspended
	}

	body := buildSubmissionBody(ctx.Record, &cfg.TaskCode, p.client.callbackTasksURL())

	slog.Info("taskv2 officer_input: dispatching to OGA portal",
		"taskId", ctx.Record.TaskID, "serviceId", cfg.ServiceID, "path", cfg.Path, "taskCode", cfg.TaskCode)

	if err := p.client.post(ctx.Context, cfg.ServiceID, cfg.Path, body); err != nil {
		return err
	}
	return ErrSuspended
}

// Render reuses the stock implementation — the schema-fetching behaviour is
// identical and our custom plugin doesn't need to override it.
func (p *OfficerInputPlugin) Render(configRaw json.RawMessage, record tfstore.TaskRecord, getTemplate tfplugins.TemplateRetriever) (map[string]any, error) {
	return tfplugins.NewOfficerInputPlugin().Render(configRaw, record, getTemplate)
}
