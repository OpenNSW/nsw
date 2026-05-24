package taskv2

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
)

type HTTPHandler struct {
	Manager *orchestrator.TaskManager
}

func NewHTTPHandler(manager *orchestrator.TaskManager) *HTTPHandler {
	return &HTTPHandler{Manager: manager}
}

// HandleGetTask returns the rendered UI payload for a single task.
//
//	GET /api/v2/tasks/{id}
func (h *HTTPHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeJSONError(w, http.StatusBadRequest, "task id is required")
		return
	}

	info, err := h.Manager.GetTaskRenderInfo(r.Context(), taskID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, info)
}

// HandleCompleteTaskStep advances a task by submitting a step payload.
//
//	POST /api/v2/tasks/{id}/step
//	body: arbitrary JSON object — passed through to the task plugin
func (h *HTTPHandler) HandleCompleteTaskStep(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if !errors.Is(err, http.ErrBodyReadAfterClose) {
			writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	// TODO(oga-callback): drop the body-id fallback once OGA POSTs directly
	// to /api/v1/tasks/{id}. For now the legacy POST /api/v1/tasks route also
	// wires here, and OGA's envelope carries task_id in the body.
	if taskID == "" {
		if id, ok := payload["task_id"].(string); ok {
			taskID = id
		}
	}
	if taskID == "" {
		writeJSONError(w, http.StatusBadRequest, "task id is required")
		return
	}

	payload = unwrapOGACallback(payload)

	if err := h.Manager.CompleteTaskStep(r.Context(), taskID, payload); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// unwrapOGACallback detects OGA's legacy TaskResponse envelope and returns the
// reviewer payload that the task plugin actually expects.
//
// OGA's sendToService posts back:
//
//	{ "task_id": "...", "workflow_id": "...", "payload": { "action": "OGA_VERIFICATION", "content": {...} } }
//
// but our plugins expect the bare reviewer form data. We detect the envelope
// (presence of task_id + workflow_id + payload.content) and lift content up.
//
// TODO(oga-callback): remove once OGA is updated to post the bare reviewer
// payload directly to /api/v1/tasks/{id} (or to a dedicated /oga-callback
// route that owns this translation).
func unwrapOGACallback(payload map[string]any) map[string]any {
	if payload == nil {
		return payload
	}
	if _, hasTaskID := payload["task_id"]; !hasTaskID {
		return payload
	}
	if _, hasWorkflowID := payload["workflow_id"]; !hasWorkflowID {
		return payload
	}
	envelope, ok := payload["payload"].(map[string]any)
	if !ok {
		return payload
	}
	content, ok := envelope["content"].(map[string]any)
	if !ok {
		return payload
	}
	slog.Info("taskv2: unwrapped OGA callback envelope", "action", envelope["action"])
	return content
}

func writeJSONResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("taskv2: failed to encode JSON response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, map[string]string{"error": message})
}
