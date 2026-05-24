package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw-task-flow/store"
	"github.com/OpenNSW/nsw/pkg/remote"
)

// ExternalReviewPlugin is our custom replacement for
// nsw-task-flow's generic_external_review plugin. It supplies the OGA portal
// with a fully-populated submission envelope.
type ExternalReviewPlugin struct {
	client *dispatchHelper
}

// NewExternalReviewPlugin builds a plugin that POSTs the trader's submitted
// form to the configured service+path with a rich body shape.
func NewExternalReviewPlugin(manager *remote.Manager, backendBaseURL string, devMode bool) *ExternalReviewPlugin {
	return &ExternalReviewPlugin{client: newDispatchHelper(manager, backendBaseURL, devMode)}
}

type externalReviewConfig struct {
	ServiceID           string `json:"service_id"`
	Path                string `json:"path"`
	ReviewerJsonFormsID string `json:"reviewer_jsonforms_id,omitempty"`
	TaskCode            string `json:"task_code,omitempty"`
}

// Execute persists the reviewer form ID + QUEUED_EXTERNALLY status, then
// POSTs the submission to the OGA portal so the officer's review queue is
// populated. The body matches the SimpleFormExternalServiceRequest shape
// used by the legacy FCAU/NPQS OGA services.
func (p *ExternalReviewPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg externalReviewConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("external_review: invalid config: %w", err)
	}
	if cfg.ServiceID == "" {
		return fmt.Errorf("external_review: service_id is required")
	}
	if cfg.Path == "" {
		return fmt.Errorf("external_review: path is required")
	}

	ctx.Record.State = "QUEUED_EXTERNALLY"

	body := buildSubmissionBody(ctx.Record, &cfg.TaskCode, p.client.callbackTasksURL())

	slog.Info("taskv2 external_review: dispatching to OGA portal",
		"taskId", ctx.Record.TaskID, "serviceId", cfg.ServiceID, "path", cfg.Path, "taskCode", cfg.TaskCode)

	if err := p.client.post(ctx.Context, cfg.ServiceID, cfg.Path, body); err != nil {
		return err
	}
	return ErrSuspended
}

// buildSubmissionBody constructs the full envelope the OGA portal expects.
func buildSubmissionBody(record *store.TaskRecord, taskCode *string, callbackURL string) map[string]any {
	if taskCode == nil || *taskCode == "" {
		taskCode = &record.ActiveTaskTemplateID
	}
	return map[string]any{
		"taskCode":   taskCode,
		"taskId":     record.TaskID,
		"workflowId": record.ParentWorkflowID,
		"serviceUrl": callbackURL,
		"data":       record.Data,
	}
}
