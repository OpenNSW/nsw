package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"

	tfplugins "github.com/OpenNSW/nsw-task-flow/plugins"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"
)

// ExternalReviewPlugin is our custom replacement for
// nsw-task-flow's generic_external_review plugin. It supplies the OGA portal
// at the configured external_url with a fully-populated submission envelope.
type ExternalReviewPlugin struct {
	client *dispatchClient
}

// NewExternalReviewPlugin builds a plugin that POSTs the trader's submitted
// form to the configured external_url with a rich body shape.
func NewExternalReviewPlugin(backendBaseURL string, devMode bool) *ExternalReviewPlugin {
	return &ExternalReviewPlugin{client: newDispatchClient(backendBaseURL, devMode)}
}

func (p *ExternalReviewPlugin) Name() string { return "generic_external_review" }

type externalReviewConfig struct {
	ExternalURL         string `json:"external_url"`
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
	if cfg.ExternalURL == "" {
		return fmt.Errorf("external_review: external_url is required")
	}

	// Mutate the in-memory record BEFORE dispatching so the body we send
	// reflects the active step (reviewer form ID, QUEUED_EXTERNALLY status).
	if cfg.ReviewerJsonFormsID != "" {
		ctx.Record.ReviewerFormID = cfg.ReviewerJsonFormsID
	}
	ctx.Record.Status = "QUEUED_EXTERNALLY"

	body := buildSubmissionBody(ctx.Record, &cfg.TaskCode, p.client.callbackTasksURL())

	slog.Info("taskv2 external_review: dispatching to OGA portal",
		"taskId", ctx.Record.TaskID, "url", cfg.ExternalURL, "taskCode", &cfg.TaskCode)

	if err := p.client.post(ctx.Context, cfg.ExternalURL, body); err != nil {
		return err
	}
	return ErrSuspended
}

// Render reuses nsw-task-flow's stock implementation — it only needs to
// surface the reviewer form schema, which our custom plugin doesn't change.
func (p *ExternalReviewPlugin) Render(configRaw json.RawMessage, record tfstore.TaskRecord, getTemplate tfplugins.TemplateRetriever) (map[string]any, error) {
	return tfplugins.NewExternalReviewPlugin(nil).Render(configRaw, record, getTemplate)
}

// buildSubmissionBody constructs the full envelope the OGA portal expects.
func buildSubmissionBody(record *tfstore.TaskRecord, taskCode *string, callbackURL string) map[string]any {
	if taskCode == nil || *taskCode == "" {
		taskCode = &record.ActiveTaskTemplateID
	}
	return map[string]any{
		"taskCode":             taskCode,
		"taskId":               record.TaskID,
		"taskType":             record.TaskType,
		"workflowId":           record.ParentWorkflowID,
		"runId":                record.ParentRunID,
		"parentNodeId":         record.ParentNodeID,
		"taskWorkflowId":       record.TaskWorkflowID,
		"subtaskNodeId":        record.SubTaskNodeID,
		"activeTaskTemplateId": record.ActiveTaskTemplateID,
		"userFormId":           record.UserFormID,
		"reviewerFormId":       record.ReviewerFormID,
		"serviceUrl":           callbackURL,
		"status":               record.Status,
		"data":                 record.Data,
	}
}
