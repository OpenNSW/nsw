package manager

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// HTTPHandler encapsulates the HTTP transport logic for TaskManager
type HTTPHandler struct {
	manager TaskManager
}

// NewHTTPHandler creates a new HTTPHandler for the task manager
func NewHTTPHandler(manager TaskManager) *HTTPHandler {
	return &HTTPHandler{manager: manager}
}

// HandleGetTask is an HTTP handler for fetching task information via GET request
func (h *HTTPHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	taskId := r.PathValue("id")
	if taskId == "" {
		writeJSONError(w, http.StatusBadRequest, "taskId is required")
		return
	}

	result, err := h.manager.GetTaskRenderInfo(r.Context(), taskId)
	if err != nil {
		// Differentiate between invalid ID/NotFound and internal errors, if necessary.
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "taskID is invalid") {
			status = http.StatusBadRequest
		} else if strings.HasPrefix(err.Error(), "task ") {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// HandleExecuteTask is an HTTP handler for executing a task via POST request
func (h *HTTPHandler) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	result, err := h.manager.ExecuteTask(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if string(err.Error()) == "task_id is required" {
			status = http.StatusBadRequest
		} else if len(err.Error()) >= 5 && string(err.Error()[:5]) == "task " {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, err.Error())
		return
	}

	// Return success response
	writeJSONResponse(w, http.StatusOK, result.ApiResponse)
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, ExecuteTaskResponse{
		Success: false,
		Error:   message,
	})
}
