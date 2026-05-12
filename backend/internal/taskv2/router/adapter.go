package router

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"
)

// TraderTaskRenderInfo is the shape the React trader-app expects from
// GET /api/v1/tasks/{id}. It is a discriminated union on `type`:
//
//	SIMPLE_FORM   — content carries traderFormInfo (+ optional ogaReviewForm,
//	                submissionResponseForm, ogaFeedback[]).
//	WAIT_FOR_EVENT — content carries display{title,description}.
//	PAYMENT       — content carries gatewayUrl, totalAmount, currency, breakdown[].
type TraderTaskRenderInfo struct {
	Type        string         `json:"type"`
	State       string         `json:"state"`
	PluginState string         `json:"pluginState"`
	Content     map[string]any `json:"content"`
}

// TraderApiResponse mirrors the wrapper plugin.ApiResponse the legacy
// trader-app HTTP handler used to write.
type TraderApiResponse struct {
	Success bool                  `json:"success"`
	Data    *TraderTaskRenderInfo `json:"data,omitempty"`
	Error   *TraderApiError       `json:"error,omitempty"`
}

type TraderApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// adaptToTraderShape transforms an nsw-task-flow TaskRecord into the
// trader-app discriminated-union shape. It pulls the plugin name and form
// schemas from the registry to construct the right Content payload.
func adaptToTraderShape(record tfstore.TaskRecord, registry *orchestrator.TaskTemplateRegistry) *TraderTaskRenderInfo {
	regEntry, _ := registry.Get(record.ActiveTaskTemplateID)
	state, pluginState := mapStatus(record.Status, regEntry.PluginName)

	switch regEntry.PluginName {
	case "generic_user_input":
		// If the previous officer round asked for changes, surface it as
		// OGA_FEEDBACK_PROVIDED so the trader-app shows the feedback banner.
		if reviewerFeedback(record.Data) != "" {
			pluginState = "OGA_FEEDBACK_PROVIDED"
		}
		return &TraderTaskRenderInfo{
			Type:        "SIMPLE_FORM",
			State:       state,
			PluginState: pluginState,
			Content:     buildSimpleFormContent(record, registry, regEntry, true /*isUserInput*/),
		}

	case "generic_officer_input":
		// Officer-only step (e.g. certificate issuance). The trader has nothing
		// to do here — render as a WAIT_FOR_EVENT so the trader-app shows a
		// "waiting for officer" view, and surface the officer's submitted form
		// once they complete the step.
		return &TraderTaskRenderInfo{
			Type:        "WAIT_FOR_EVENT",
			State:       state,
			PluginState: pluginState,
			Content:     buildWaitForEventContent(record, registry, regEntry),
		}

	case "generic_external_review":
		// Officer is reviewing the trader's submission. The trader needs to
		// see what they submitted (read-only) while waiting — render the user
		// form in SIMPLE_FORM shape with pluginState=OGA_ACKNOWLEDGED, which
		// the trader-app SimpleForm component already treats as read-only.
		return &TraderTaskRenderInfo{
			Type:        "SIMPLE_FORM",
			State:       state,
			PluginState: pluginState,
			Content:     buildSimpleFormContent(record, registry, regEntry, true /*isUserInput*/),
		}

	case "register_task_and_wait":
		return &TraderTaskRenderInfo{
			Type:        "WAIT_FOR_EVENT",
			State:       state,
			PluginState: pluginState,
			Content:     buildWaitForEventContent(record, registry, regEntry),
		}

	case "generic_payment":
		return &TraderTaskRenderInfo{
			Type:        "PAYMENT",
			State:       state,
			PluginState: pluginState,
			Content:     buildPaymentContent(record, regEntry),
		}
	case "generic_api_call":
		return &TraderTaskRenderInfo{
			Type:        "FIRE_AND_FORGET",
			State:       state,
			PluginState: pluginState,
			Content:     buildFireAndForgetContent(record, registry, regEntry),
		}
	default:
		// Unknown plugin: surface raw record fields under a generic SIMPLE_FORM.
		slog.Warn("taskv2 adapter: unknown plugin, returning generic shape",
			"plugin", regEntry.PluginName, "taskId", record.TaskID)
		return &TraderTaskRenderInfo{
			Type:        "SIMPLE_FORM",
			State:       state,
			PluginState: pluginState,
			Content: map[string]any{
				"traderFormInfo": map[string]any{
					"title":    "Task " + record.TaskID,
					"schema":   map[string]any{"type": "object"},
					"uiSchema": map[string]any{"type": "VerticalLayout", "elements": []any{}},
					"formData": record.Data,
				},
			},
		}
	}

}

// mapStatus translates the nsw-task-flow record status string into the
// trader-app's (taskState, pluginState) pair. The trader-app's plugin
// components branch on these to decide which UI to show, so the mapping
// must use values the React code actually handles.
func mapStatus(recordStatus, plugin string) (state, pluginState string) {
	switch recordStatus {
	case "":
		return "INITIALIZED", "INITIALIZED"
	case "STARTING":
		return "INITIALIZED", "INITIALIZED"
	case "PENDING_USER":
		return "IN_PROGRESS", "INITIALIZED"
	case "QUEUED_EXTERNALLY":
		// generic_external_review is a SimpleForm-with-callback in trader-app
		// terms, so it shows OGA_ACKNOWLEDGED while the trader waits.
		if plugin == "generic_external_review" {
			return "IN_PROGRESS", "OGA_ACKNOWLEDGED"
		}
		return "IN_PROGRESS", "NOTIFIED_SERVICE"
	case "WAITING_FOR_EVENT":
		return "IN_PROGRESS", "NOTIFIED_SERVICE"
	case "PENDING_PAYMENT":
		return "IN_PROGRESS", "IDLE"
	case "DISPATCHED":
		return "COMPLETED", "RECEIVED_CALLBACK"
	case "COMPLETED":
		return "COMPLETED", "COMPLETED"
	case "FAILED":
		return "FAILED", "SUBMISSION_FAILED"
	default:
		return "IN_PROGRESS", recordStatus
	}
}

// buildSimpleFormContent renders SIMPLE_FORM content from a generic_user_input
// or generic_officer_input task. It looks up the JSONForms schema by
// UserFormID/ReviewerFormID in the registry's generic templates and copies
// any prior-round trader form data into formData (so the UI can pre-fill).
func buildSimpleFormContent(
	record tfstore.TaskRecord,
	registry *orchestrator.TaskTemplateRegistry,
	_ orchestrator.TaskTemplateEntry,
	isUserInput bool,
) map[string]any {
	formID := record.UserFormID
	if !isUserInput {
		formID = record.ReviewerFormID
	}

	formSchema := lookupFormSchema(registry, formID)
	formData := extractNamespacedFormData(record.Data, "userform")
	if formData == nil {
		formData = fallbackFormData(record.Data)
	}
	if formData == nil {
		formData = map[string]any{}
	}

	content := map[string]any{
		"traderFormInfo": map[string]any{
			"title":    formSchema["title"],
			"schema":   formSchema["schema"],
			"uiSchema": formSchema["uiSchema"],
			"formData": formData,
		},
	}

	// Surface the reviewer form schema only after the OGA officer has actually
	// submitted a review. We key off the presence of "reviewerform" in
	// record.Data — that key is only populated by the OGA callback flow
	// (router.normalizePayload → mergeOGAReview). Before that, the trader
	// would see an empty reviewer card while still waiting for review, so we
	// omit the field entirely until there's data to display.
	if record.ReviewerFormID != "" && record.ReviewerFormID != formID {
		if reviewerData := extractNamespacedFormData(record.Data, "reviewerform"); len(reviewerData) > 0 {
			if ogaSchema := lookupFormSchema(registry, record.ReviewerFormID); ogaSchema != nil {
				content["ogaReviewForm"] = map[string]any{
					"title":    ogaSchema["title"],
					"schema":   ogaSchema["schema"],
					"uiSchema": ogaSchema["uiSchema"],
					"formData": reviewerData,
				}
			}
		}
	}

	// If the officer requested changes (review_outcome=needs_more_info or
	// rejection_reason populated), surface it as a single-entry ogaFeedback so
	// the trader-app's LatestFeedbackBanner / OGAFeedbackHistory can render.
	if feedback := reviewerFeedback(record.Data); feedback != "" {
		content["ogaFeedback"] = []map[string]any{
			{
				"content":   map[string]any{"feedback": feedback},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"round":     1,
			},
		}
	}

	return content
}

// reviewerFeedback returns the latest officer-supplied feedback text, or "" if
// no change-request is currently pending. We treat "needs_more_info" with a
// rejection_reason as a feedback request; other outcomes (approve / reject)
// are terminal and don't surface feedback to the trader.
func reviewerFeedback(data map[string]any) string {
	reviewer, ok := data["reviewerform"].(map[string]any)
	if !ok {
		return ""
	}
	outcome, _ := reviewer["review_outcome"].(string)
	if outcome != "needs_more_info" {
		return ""
	}
	feedback, _ := reviewer["rejection_reason"].(string)
	return feedback
}

// buildWaitForEventContent renders WAIT_FOR_EVENT content. nsw-task-flow's
// register_task_and_wait / generic_external_review plugins don't carry display
// strings, so we fall back to a sensible default derived from the task type.
func buildWaitForEventContent(
	record tfstore.TaskRecord,
	registry *orchestrator.TaskTemplateRegistry,
	regEntry orchestrator.TaskTemplateEntry,
) map[string]any {
	// Default display strings keyed by record status.
	titles := map[string]string{
		"waiting":   "Waiting for external event",
		"failed":    "External notification failed",
		"completed": "External event received",
	}
	descriptions := map[string]string{
		"waiting":   "This step is queued externally. You will be notified when the next action is required.",
		"failed":    "Could not deliver the notification. Please retry.",
		"completed": "The external system has responded. The workflow will continue.",
	}

	// If the plugin properties carry override text, prefer those.
	type pluginDisplay struct {
		Display struct {
			Title       any `json:"title"`
			Description any `json:"description"`
		} `json:"display"`
	}
	if len(regEntry.PluginProperties) > 0 {
		var pd pluginDisplay
		if err := json.Unmarshal(regEntry.PluginProperties, &pd); err == nil {
			if t, ok := pd.Display.Title.(map[string]any); ok {
				for k, v := range t {
					if s, ok := v.(string); ok {
						titles[k] = s
					}
				}
			} else if s, ok := pd.Display.Title.(string); ok {
				titles["waiting"] = s
				titles["completed"] = s
			}
			if d, ok := pd.Display.Description.(map[string]any); ok {
				for k, v := range d {
					if s, ok := v.(string); ok {
						descriptions[k] = s
					}
				}
			} else if s, ok := pd.Display.Description.(string); ok {
				descriptions["waiting"] = s
				descriptions["completed"] = s
			}
		}
	}

	content := map[string]any{
		"display": map[string]any{
			"title":       titles,
			"description": descriptions,
		},
	}

	// Include trader-submitted data as a read-only form if available.
	if record.UserFormID != "" {
		if schema := lookupFormSchema(registry, record.UserFormID); schema != nil {
			formData := extractNamespacedFormData(record.Data, "userform")
			if formData == nil {
				formData = fallbackFormData(record.Data)
			}
			content["submittedForm"] = map[string]any{
				"title":    schema["title"],
				"schema":   schema["schema"],
				"uiSchema": schema["uiSchema"],
				"formData": formData,
			}
		}
	}

	// Include reviewer response form when available (shown after callback).
	if record.ReviewerFormID != "" {
		if schema := lookupFormSchema(registry, record.ReviewerFormID); schema != nil {
			content["eventReviewForm"] = map[string]any{
				"title":    schema["title"],
				"schema":   schema["schema"],
				"uiSchema": schema["uiSchema"],
				"formData": extractNamespacedFormData(record.Data, "reviewerform"),
			}
		}
	}

	return content
}

// buildFireAndForgetContent renders FIRE_AND_FORGET content. It surfaces the
// destination URL from plugin properties and the previously submitted userform
// data with its schema so the trader-app can show what was dispatched and where.
func buildFireAndForgetContent(
	record tfstore.TaskRecord,
	_ *orchestrator.TaskTemplateRegistry,
	regEntry orchestrator.TaskTemplateEntry,
) map[string]any {
	content := map[string]any{}

	type apiCallProps struct {
		URL string `json:"url"`
	}
	if len(regEntry.PluginProperties) > 0 {
		var props apiCallProps
		if err := json.Unmarshal(regEntry.PluginProperties, &props); err == nil && props.URL != "" {
			content["endpoint"] = props.URL
		}
	}

	// Pass all workflow-level data that was dispatched to the external API,
	// stripping internal bookkeeping keys and form namespaces (those are only
	// meaningful to the workflow engine, not the trader).
	if dispatched := fallbackFormData(record.Data); len(dispatched) > 0 {
		content["data"] = dispatched
	}

	return content
}

// buildPaymentContent renders PAYMENT content. nsw-task-flow's generic_payment
// plugin only stores a service URL — there's no breakdown/totalAmount baked in.
// For the demo we surface a single "Phytosanitary Certificate Fee" line item;
// production deployments would expand this from real config.
func buildPaymentContent(_ tfstore.TaskRecord, _ orchestrator.TaskTemplateEntry) map[string]any {
	return map[string]any{
		"gatewayUrl":  "",
		"totalAmount": 1500,
		"currency":    "LKR",
		"breakdown": []map[string]any{
			{
				"description": "Phytosanitary Certificate Processing Fee",
				"category":    "ADDITION",
				"type":        "FIXED",
				"quantity":    1,
				"unitPrice":   1500,
				"amount":      1500,
			},
		},
	}
}

// lookupFormSchema fetches a generic JSON template (a JSONForms schema) by ID
// and returns its {title, schema, uiSchema} fields. The npqs/* files use
// "uischema" (lowercase, no separator) — this normalises to "uiSchema".
func lookupFormSchema(registry *orchestrator.TaskTemplateRegistry, formID string) map[string]any {
	if formID == "" {
		return nil
	}
	raw, ok := registry.GetGenericTemplate(formID)
	if !ok {
		slog.Warn("taskv2 adapter: form template not found", "formId", formID)
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		slog.Warn("taskv2 adapter: form template invalid JSON", "formId", formID, "error", err)
		return nil
	}
	out := map[string]any{
		"title":  decoded["title"],
		"schema": decoded["schema"],
	}
	if v, ok := decoded["uiSchema"]; ok {
		out["uiSchema"] = v
	} else if v, ok := decoded["uischema"]; ok {
		out["uiSchema"] = v
	}
	if out["title"] == nil {
		out["title"] = formID
	}
	return out
}

// extractNamespacedFormData reads a top-level namespaced map from record.Data
// (e.g. record.Data["userform"]) and returns it as a plain map. Returns nil
// if the namespace is absent.
func extractNamespacedFormData(data map[string]any, namespace string) map[string]any {
	if data == nil {
		return nil
	}
	v, ok := data[namespace]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

// fallbackFormData returns a best-effort form payload when userform is not namespaced.
func fallbackFormData(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range data {
		switch k {
		case "_task_id", "reviewerform", "userform":
			continue
		default:
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
