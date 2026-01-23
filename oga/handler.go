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
	service OGAService
}

// NewOGAHandler creates a new OGA handler instance
func NewOGAHandler(service OGAService) *OGAHandler {
	return &OGAHandler{
		service: service,
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
// Returns all applications ready for OGA review
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
// Returns a specific application by task ID
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

// HandleTaskCompleted handles POST /api/oga/tasks/{taskId}/completed
// Called by Task Manager when a task is completed/rejected to remove it from OGA list
func (h *OGAHandler) HandleTaskCompleted(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	taskID, err := h.parseTaskID(w, r)
	if err != nil {
		return
	}

	ctx := r.Context()
	if err := h.service.RemoveApplication(ctx, taskID); err != nil {
		slog.WarnContext(ctx, "failed to remove application from list",
			"taskID", taskID,
			"error", err)
		// Return 404 if not found, but usually it's fine
	}

	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{"success": true})
}

// HandleNotification handles POST /api/oga/notifications
// Receives notifications from Task Manager when applications are ready for review
func (h *OGAHandler) HandleNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var notification OGATaskNotification

	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx := r.Context()

	// Add application to service
	if err := h.service.AddApplication(ctx, notification); err != nil {
		slog.ErrorContext(ctx, "failed to add application",
			"taskID", notification.TaskID,
			"error", err)
		utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to add application")
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Application added for review",
	})
}
