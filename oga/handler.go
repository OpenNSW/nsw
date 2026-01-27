package oga

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
		WriteJSONError(w, http.StatusBadRequest, "taskId is required")
		return uuid.Nil, errors.New("taskId is required")
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid taskId format")
		return uuid.Nil, err
	}
	return taskID, nil
}

// HandleInjectData handles POST /api/oga/inject
// This is the endpoint that external services use to inject data into OGA portal
func (h *OGAHandler) HandleInjectData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()

	var req InjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Create application in database
	if err := h.service.CreateApplication(ctx, &req); err != nil {
		slog.ErrorContext(ctx, "failed to create application", "error", err)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to create application: "+err.Error())
		return
	}

	slog.InfoContext(ctx, "data injected successfully",
		"taskID", req.TaskID,
		"consignmentID", req.ConsignmentID)

	WriteJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Data injected successfully",
		"taskId":  req.TaskID,
	})
}

// HandleGetApplications handles GET /api/oga/applications
// Returns all applications, optionally filtered by status query parameter
func (h *OGAHandler) HandleGetApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()
	status := r.URL.Query().Get("status")

	applications, err := h.service.GetApplications(ctx, status)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get applications", "error", err)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to get applications")
		return
	}

	WriteJSONResponse(w, http.StatusOK, applications)
}

// HandleGetApplication handles GET /api/oga/applications/{taskId}
// Returns a specific application by task ID
func (h *OGAHandler) HandleGetApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
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
			WriteJSONError(w, http.StatusNotFound, "Application not found")
		} else {
			slog.ErrorContext(ctx, "failed to get application",
				"taskID", taskID,
				"error", err)
			WriteJSONError(w, http.StatusInternalServerError, "Failed to get application")
		}
		return
	}

	WriteJSONResponse(w, http.StatusOK, application)
}

// HandleHealth handles GET /health
// Simple health check endpoint
func (h *OGAHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "oga-portal",
	})
}

// HandleReviewApplication handles POST /api/oga/applications/{taskId}/review
// Called when OGA officer approves/rejects an application
// Sends the response back to the originating service
func (h *OGAHandler) HandleReviewApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	taskID, err := h.parseTaskID(w, r)
	if err != nil {
		return
	}

	ctx := r.Context()

	// Parse request body
	var requestBody struct {
		Decision      string `json:"decision"`      // "APPROVED" or "REJECTED"
		ReviewerNotes string `json:"reviewerNotes"` // Optional notes
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate decision
	if requestBody.Decision != "APPROVED" && requestBody.Decision != "REJECTED" {
		WriteJSONError(w, http.StatusBadRequest, "Decision must be either APPROVED or REJECTED")
		return
	}

	// Process review and send response to service
	if err := h.service.ReviewApplication(ctx, taskID, requestBody.Decision, requestBody.ReviewerNotes); err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			WriteJSONError(w, http.StatusNotFound, "Application not found")
		} else {
			slog.ErrorContext(ctx, "failed to review application",
				"taskID", taskID,
				"error", err)
			WriteJSONError(w, http.StatusInternalServerError, "Failed to review application: "+err.Error())
		}
		return
	}

	slog.InfoContext(ctx, "application reviewed",
		"taskID", taskID,
		"decision", requestBody.Decision)

	WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Application reviewed successfully",
	})
}
