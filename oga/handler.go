package oga

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/utils"
	"github.com/google/uuid"
)

// OGAHandler handles HTTP requests for OGA portal operations
type OGAHandler struct {
	service       OGAService
	backendClient BackendClient
}

// NewOGAHandler creates a new OGA handler instance
func NewOGAHandler(service OGAService, backendClient BackendClient) *OGAHandler {
	return &OGAHandler{
		service:       service,
		backendClient: backendClient,
	}
}

// parseTaskID extracts and parses the taskId from the request path
func (h *OGAHandler) parseTaskID(w http.ResponseWriter, r *http.Request) (uuid.UUID, error) {
	taskIDStr := r.PathValue("taskId")
	if taskIDStr == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "taskId is required")
		return uuid.Nil, errors.New("taskId is required")
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid taskId format")
		return uuid.Nil, err
	}
	return taskID, nil
}

// HandleGetApplications handles GET /api/oga/applications
// Returns all applications ready for OGA review (from OGA's own database)
func (h *OGAHandler) HandleGetApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()
	applications, err := h.service.GetApplications(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get applications", "error", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to get applications")
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, applications)
}

// HandleGetApplication handles GET /api/oga/applications/{taskId}
// Returns a specific application by task ID (from OGA's own database)
func (h *OGAHandler) HandleGetApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	taskID, err := h.parseTaskID(w, r)
	if err != nil {
		return
	}

	ctx := r.Context()
	application, err := h.service.GetApplication(ctx, taskID)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			utils.WriteJSONError(w, http.StatusNotFound, "Application not found")
		} else {
			slog.ErrorContext(ctx, "failed to get application",
				"taskID", taskID,
				"error", err)
			utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to get application")
		}
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, application)
}

// HandleApproveApplication handles POST /api/oga/applications/{taskId}/approve
// Called when OGA officer approves/rejects an application - calls backend POST /api/tasks/{taskId}
func (h *OGAHandler) HandleApproveApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	taskID, err := h.parseTaskID(w, r)
	if err != nil {
		return
	}

	ctx := r.Context()

	// Parse request body to get OGA form data
	var requestBody struct {
		Decision string                 `json:"decision"` // "APPROVED" or "REJECTED"
		Data     map[string]interface{} `json:"data"`     // OGA form submission data
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Prepare payload for backend
	payload := &ExecutionPayload{
		Action: "OGA_VERIFICATION",
		Content: map[string]interface{}{
			"decision": requestBody.Decision,
			"data":     requestBody.Data,
		},
	}

	// Call backend POST /api/tasks/{taskId}
	if err := h.backendClient.ExecuteTask(ctx, taskID, payload); err != nil {
		slog.ErrorContext(ctx, "failed to execute task in backend",
			"taskID", taskID,
			"error", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to approve application: "+err.Error())
		return
	}

	slog.InfoContext(ctx, "application processed",
		"taskID", taskID,
		"decision", requestBody.Decision)

	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Application processed successfully",
	})
}

// HandleApprove is an alias for HandleApproveApplication
func (h *OGAHandler) HandleApprove(w http.ResponseWriter, r *http.Request) {
	h.HandleApproveApplication(w, r)
}



