package form

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/utils"
	"github.com/google/uuid"
)

// FormHandler handles HTTP requests for form operations
type FormHandler struct {
	service FormService
}

// NewFormHandler creates a new FormHandler instance
func NewFormHandler(service FormService) *FormHandler {
	return &FormHandler{
		service: service,
	}
}

// GetFormByID handles GET /api/forms/:formId
// Returns the JSON Forms schema that portals can directly use
func (h *FormHandler) GetFormByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract formID (UUID) from URL path
	// Expected format: /api/forms/{formId}
	formIDStr := r.PathValue("formId")
	if formIDStr == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "formId is required")
		return
	}

	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid formId format")
		return
	}

	ctx := r.Context()
	form, err := h.service.GetFormByID(ctx, formID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to retrieve form",
			"formID", formID,
			"error", err)
		if errors.Is(err, ErrFormNotFound) {
			utils.WriteJSONError(w, http.StatusNotFound, "Form not found")
		} else {
			utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve form")
		}
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, form)
}

// GetFormByTaskID handles GET /api/tasks/:taskId/form
// Returns the JSON Forms schema for a task (extracts formId from Task.Config)
// Portals can call this with just a taskId without needing to know the formId
func (h *FormHandler) GetFormByTaskID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	taskIDStr := r.PathValue("taskId")
	if taskIDStr == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, "taskId is required")
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, "invalid taskId format")
		return
	}

	ctx := r.Context()
	form, err := h.service.GetFormByTaskID(ctx, taskID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to retrieve form by taskId",
			"taskID", taskID,
			"error", err)
		if errors.Is(err, ErrTaskNotFound) {
			utils.WriteJSONError(w, http.StatusNotFound, "Form not found for the given task")
		} else {
			utils.WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve form")
		}
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, form)
}
