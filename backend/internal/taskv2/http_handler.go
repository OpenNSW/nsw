package taskv2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"

	"github.com/OpenNSW/nsw/internal/taskv2/renderer"
)

// TaskFetcher is the narrow surface HandleGetTask needs from the task store.
type TaskFetcher interface {
	GetTask(ctx context.Context, taskID string) (tfstore.TaskRecord, bool)
}

type HTTPHandler struct {
	Manager   *orchestrator.TaskManager
	Store     TaskFetcher
	Assembler *renderer.ZoneViewAssembler
}

func NewHTTPHandler(manager *orchestrator.TaskManager, store TaskFetcher, assembler *renderer.ZoneViewAssembler) *HTTPHandler {
	return &HTTPHandler{Manager: manager, Store: store, Assembler: assembler}
}

// HandleGetTask returns the ZoneView payload for a single task.
//
//	GET /api/v1/tasks/{id}
func (h *HTTPHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	// TODO: retrieve the authenticated context and validate it against the
	// task's ownership bounds before returning ZoneView.
	taskID := r.PathValue("id")
	if taskID == "" {
		writeJSONError(w, http.StatusBadRequest, "task id is required")
		return
	}

	record, ok := h.Store.GetTask(r.Context(), taskID)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}

	zv, err := h.Assembler.Assemble(r.Context(), record)
	if err != nil {
		slog.Error("taskv2: failed to assemble zone view", "taskId", taskID, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "An internal error occurred while loading the task")
		return
	}

	writeJSONResponse(w, http.StatusOK, zv)
}

// HandleCompleteTaskStep advances a task by submitting a step payload.
//
//	POST /api/v1/tasks/{id}
//	body: arbitrary JSON object — passed through to the task plugin
func (h *HTTPHandler) HandleCompleteTaskStep(w http.ResponseWriter, r *http.Request) {
	// TODO: retrieve the authenticated context and validate it against the
	// task's ownership bounds before completing the step.
	taskID := r.PathValue("id")

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// An empty body is a valid acknowledge-style completion; only fail on
		// genuinely malformed JSON.
		if !errors.Is(err, io.EOF) && !errors.Is(err, http.ErrBodyReadAfterClose) {
			writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			slog.Error("taskv2: failed to decode request body", "error", err)
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
		slog.Error("taskv2: missing task id in request")
		return
	}

	payload = unwrapOGACallback(payload)

	if err := h.Manager.CompleteTaskStep(r.Context(), taskID, payload); err != nil {
		slog.Error("taskv2: failed to complete task step", "taskId", taskID, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "An internal error occurred while processing the task")
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
