// Package router exposes nsw-task-flow's TaskManager over HTTP.
//
// New endpoints (preferred):
//
//	POST /api/v1/start-workflow     — start a parent workflow by template ID
//	GET  /api/v1/tasks              — list every task with render info
//	GET  /api/v1/tasks/{id}         — render info for one task
//	POST /api/v1/tasks/{id}         — submit data and resume the active step
//
// Legacy compatibility:
//
//	POST /api/v1/tasks              — accepts the old {task_id, payload:{action,content}}
//	                                  body shape and forwards to CompleteTaskStep,
//	                                  letting trader-app screens still drive the
//	                                  flow without rewrites.
package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	engine "github.com/OpenNSW/go-temporal-workflow"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw-task-flow/store"

	"github.com/google/uuid"
)

// Router wraps the nsw-task-flow TaskManager and parent workflow manager
// behind an HTTP transport.
type Router struct {
	tm       *orchestrator.TaskManager
	parent   engine.TemporalManager
	registry *orchestrator.TaskTemplateRegistry
}

func New(tm *orchestrator.TaskManager, parent engine.TemporalManager, registry *orchestrator.TaskTemplateRegistry) *Router {
	return &Router{tm: tm, parent: parent, registry: registry}
}

// Manager exposes the underlying TaskManager (used by the OGA router so it can
// share the same db/registry through CompleteTaskStep).
func (rt *Router) Manager() *orchestrator.TaskManager { return rt.tm }

// Registry exposes the template registry (used by the OGA router).
func (rt *Router) Registry() *orchestrator.TaskTemplateRegistry { return rt.registry }

// ── Handlers ──────────────────────────────────────────────────────────────────

// StartWorkflowRequest matches the demo's POST /api/start body, but generalised:
// the caller supplies a `workflow_template_id` (registered in the TaskTemplateRegistry)
// and an optional initial-variables map.
type StartWorkflowRequest struct {
	WorkflowTemplateID string         `json:"workflow_template_id"`
	WorkflowID         string         `json:"workflow_id,omitempty"`
	InitialVariables   map[string]any `json:"initial_variables,omitempty"`
}

type StartWorkflowResponse struct {
	Status     string `json:"status"`
	WorkflowID string `json:"workflow_id"`
}

// HandleStartWorkflow starts a parent workflow from a registered template.
func (rt *Router) HandleStartWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req StartWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.WorkflowTemplateID == "" {
		writeError(w, http.StatusBadRequest, "workflow_template_id is required")
		return
	}

	def, ok := rt.registry.GetWorkflow(req.WorkflowTemplateID)
	if !ok {
		writeError(w, http.StatusNotFound,
			fmt.Sprintf("workflow_template_id %q is not registered", req.WorkflowTemplateID))
		return
	}

	workflowID := req.WorkflowID
	if workflowID == "" {
		workflowID = fmt.Sprintf("%s-%s", req.WorkflowTemplateID, uuid.New().String()[:8])
	}
	initial := req.InitialVariables
	if initial == nil {
		initial = map[string]any{}
	}
	initial["_started_at"] = time.Now().UTC().Format(time.RFC3339)

	if err := rt.parent.StartWorkflow(r.Context(), workflowID, def, initial); err != nil {
		slog.Error("taskv2 router: failed to start parent workflow",
			"workflowId", workflowID, "templateId", req.WorkflowTemplateID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to start workflow: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, StartWorkflowResponse{Status: "ok", WorkflowID: workflowID})
}

// HandleListTasks returns render info for every TaskRecord in the store.
func (rt *Router) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	writeJSON(w, http.StatusOK, rt.tm.GetAllTasksRenderInfo())
}

// HandleGetTask returns render info for one task by ID, in the
// trader-app's discriminated-union shape ({type, state, pluginState, content}).
//
// Query param `raw=1` returns the unwrapped nsw-task-flow render info instead
// — useful for debugging/curl callers that don't want the trader-app shim.
func (rt *Router) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "taskId is required")
		return
	}

	resolvedID := taskID
	if r.URL.Query().Get("raw") == "1" {
		if resolved, ok := rt.resolveTaskID(taskID); ok {
			resolvedID = resolved
		} else {
			writeError(w, http.StatusNotFound, "task "+taskID+" not found")
			return
		}
		info, err := rt.tm.GetTaskRenderInfo(resolvedID)
		if err != nil {
			httpStatusForErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, info)
		return
	}

	record, ok := rt.tm.GetTask(resolvedID)
	if !ok {
		if resolved, ok := rt.findTaskByNodeRef(taskID); ok {
			record = resolved
		} else {
			writeJSON(w, http.StatusNotFound, TraderApiResponse{
				Success: false,
				Error:   &TraderApiError{Code: "TASK_NOT_FOUND", Message: "task " + taskID + " not found"},
			})
			return
		}
	}

	rendered := adaptToTraderShape(record, rt.registry)
	writeJSON(w, http.StatusOK, TraderApiResponse{Success: true, Data: rendered})
}

func httpStatusForErr(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "not found") {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

// HandleCompleteTaskStep submits the namespaced form payload for the active step.
//
// Accepts three body shapes and normalises them to the namespaced map that
// nsw-task-flow's CompleteTaskStep expects:
//
//  1. Modern (curl):       {"userform": {...}} or {"reviewerform": {...}}
//  2. Trader-app legacy:   {"task_id":"...", "workflow_id":"...", "payload":{"action":"SUBMIT_FORM","content":{...}}}
//  3. Bare form data:      {"field1":"...", "field2":"..."} — wrapped under "userform"
//
// Action-driven namespacing for the legacy shape:
//
//	SUBMIT_FORM, SAVE_AS_DRAFT     →  wrap content under "userform"
//	OGA_VERIFICATION               →  wrap content under "reviewerform"
//	INITIATE_PAYMENT, PAYMENT_*    →  treated as namespaced as-is
//	(other actions)                →  default to "userform"
//
// Without correct namespacing, the workflow's input/output_mapping (which
// references "userform" / "reviewerform") wouldn't see the data.
func (rt *Router) HandleCompleteTaskStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	taskID := r.PathValue("id")

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// Pull task_id out of legacy envelope if needed.
	if taskID == "" {
		if v, ok := raw["task_id"].(string); ok {
			taskID = v
		}
	}
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task id required (path or body.task_id)")
		return
	}

	resolvedID, ok := rt.resolveTaskID(taskID)
	if !ok {
		writeError(w, http.StatusNotFound, "task "+taskID+" not found")
		return
	}

	// INITIATE_PAYMENT is a "I'm about to pay" notification from the trader-app.
	// Our generic_payment plugin already dispatched and is waiting for the
	// final PAYMENT_SUCCESS / PAYMENT_FAILED callback — completing the task
	// step now would advance the workflow with empty data and fail downstream
	// output_mapping. Acknowledge and return without resuming the activity.
	if pl, ok := raw["payload"].(map[string]any); ok {
		if action, _ := pl["action"].(string); strings.EqualFold(action, "INITIATE_PAYMENT") {
			writeJSON(w, http.StatusOK, map[string]any{
				"success": true,
				"data":    map[string]string{"task_id": resolvedID},
			})
			return
		}
	}

	payload := normalizePayload(raw)

	if err := rt.tm.CompleteTaskStep(r.Context(), resolvedID, payload); err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			writeError(w, http.StatusNotFound, err.Error())
		case strings.Contains(err.Error(), "already completed"):
			writeError(w, http.StatusConflict, err.Error())
		default:
			slog.Error("taskv2 router: complete step failed", "taskId", resolvedID, "error", err)
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]string{"task_id": resolvedID},
	})
}

// normalizePayload transforms an incoming HTTP body into the namespaced map
// that nsw-task-flow's CompleteTaskStep merges into record.Data. See
// HandleCompleteTaskStep's doc comment for the supported shapes.
//
// OGA_VERIFICATION is published in two shapes simultaneously: under
// "reviewerform" (so application-style steps with reviewerform.* output_mapping
// can extract values) and at the top level (so wait-for-event steps with flat
// output_mapping like "lab_result" → ... can extract them too). This avoids
// having to teach the OGA portal which namespace each task code expects.
//
// OGA_VERIFICATION_FEEDBACK is mapped to a needs_more_info reviewerform, which
// is the workflow shape that drives the review_gateway loopback in our NPQS
// templates. The original feedback text is carried as rejection_reason.
func normalizePayload(raw map[string]any) map[string]any {
	// Modern shape: already namespaced ({userform: {...}} / {reviewerform: {...}})
	if _, ok := raw["userform"]; ok {
		return raw
	}
	if _, ok := raw["reviewerform"]; ok {
		return raw
	}

	// Legacy shape: {task_id, workflow_id, payload:{action, content}}
	if pl, ok := raw["payload"].(map[string]any); ok {
		action, _ := pl["action"].(string)
		content, _ := pl["content"].(map[string]any)
		if content == nil {
			content = map[string]any{}
		}
		switch strings.ToUpper(action) {
		case "SUBMIT_FORM", "SAVE_AS_DRAFT":
			return map[string]any{"userform": content}
		case "OGA_VERIFICATION":
			return mergeOGAReview(content)
		case "OGA_VERIFICATION_FEEDBACK":
			feedback, _ := content["feedback"].(string)
			reviewer := map[string]any{"review_outcome": "needs_more_info"}
			if feedback != "" {
				reviewer["rejection_reason"] = feedback
			}
			return map[string]any{
				"reviewerform":      reviewer,
				"doc_review_result": "needs_more_info",
			}
		case "PAYMENT_SUCCESS":
			ref, _ := content["payment_reference_number"].(string)
			if ref == "" {
				ref = "PAY-" + uuid.New().String()[:8]
			}
			return map[string]any{
				"payment_status":           "success",
				"payment_reference_number": ref,
			}
		case "PAYMENT_FAILED":
			ref, _ := content["payment_reference_number"].(string)
			return map[string]any{
				"payment_status":           "fail",
				"payment_reference_number": ref,
			}
		case "":
			// No action header — best-effort wrap under userform.
			return map[string]any{"userform": content}
		default:
			// Unknown action — pass through as-is.
			slog.Info("taskv2 router: passthrough action", "action", action)
			return content
		}
	}

	// Bare body — strip envelope fields and wrap the rest under userform.
	stripped := map[string]any{}
	for k, v := range raw {
		switch k {
		case "task_id", "workflow_id":
			continue
		default:
			stripped[k] = v
		}
	}
	if len(stripped) == 0 {
		return map[string]any{}
	}
	return map[string]any{"userform": stripped}
}

// mergeOGAReview returns the reviewer's content both at the top level and
// nested under "reviewerform". Sub-workflow output_mappings in the NPQS
// templates use both shapes — application reviews read reviewerform.review_outcome,
// while wait-for-event steps read flat keys like lab_result, visual_result, etc.
// doc_review_result is always derived from review_outcome so the shipping-docs
// sub-workflow's output_mapping can find it in the task result.
func mergeOGAReview(content map[string]any) map[string]any {
	out := map[string]any{"reviewerform": content}
	for k, v := range content {
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = v
	}
	if _, has := out["doc_review_result"]; !has {
		if outcome, ok := content["review_outcome"].(string); ok && outcome != "" {
			out["doc_review_result"] = outcome
		}
	}
	return out
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("taskv2 router: encode JSON failed", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"success": false, "error": msg})
}

// Discard unused import warning protection in case the file is built without
// every helper being referenced — context is used by HandleCompleteTaskStep
// indirectly via r.Context().
var _ = context.Background

func (rt *Router) resolveTaskID(taskID string) (string, bool) {
	if taskID == "" {
		return "", false
	}
	if _, ok := rt.tm.GetTask(taskID); ok {
		return taskID, true
	}
	if rec, ok := rt.findTaskByNodeRef(taskID); ok {
		return rec.TaskID, true
	}
	return "", false
}

func (rt *Router) findTaskByNodeRef(nodeRef string) (store.TaskRecord, bool) {
	if nodeRef == "" {
		return store.TaskRecord{}, false
	}
	var nodeID string
	var suffix string
	if strings.Contains(nodeRef, ":") {
		parts := strings.SplitN(nodeRef, ":", 2)
		nodeID = parts[0]
		suffix = parts[1]
	}
	for _, rec := range rt.tm.GetAllTasks() {
		if rec.SubTaskNodeID == nodeRef || rec.ParentNodeID == nodeRef {
			return rec, true
		}
		if nodeID != "" {
			if rec.SubTaskNodeID == nodeID || rec.ParentNodeID == nodeID {
				if suffix == "" || suffix == rec.ParentWorkflowID || suffix == rec.TaskWorkflowID || suffix == rec.TaskID {
					return rec, true
				}
			}
		}
	}
	return store.TaskRecord{}, false
}
