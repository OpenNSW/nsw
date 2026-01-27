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


