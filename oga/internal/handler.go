package internal

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OGAHandler handles HTTP requests for OGA portal operations
type OGAHandler struct {
	service    OGAService
	backendURL string
	httpClient *http.Client
}

func NewOGAHandler(service OGAService, backendURL string) *OGAHandler {
	return &OGAHandler{
		service:    service,
		backendURL: backendURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // It's also good practice to set a timeout.
		},
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
		"workflowID", req.WorkflowID)

	WriteJSONResponse(w, http.StatusCreated, map[string]any{
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
	page, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "Invalid page number")
	}
	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))

	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "Invalid page size")
	}

	result, err := h.service.GetApplications(ctx, status, page, pageSize)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get applications", "error", err)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to get applications")
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
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
	WriteJSONResponse(w, http.StatusOK, map[string]any{
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
	var requestBody map[string]any

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Process review and send response to service
	if err := h.service.ReviewApplication(ctx, taskID, requestBody); err != nil {
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
	)

	WriteJSONResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Application reviewed successfully",
	})
}

// HandleGetUploadURL forwards the secure file download request to the NSW Backend.
// It retrieves the short-lived presigned URL, avoiding exposing the unauthenticated `/content` route.
func (h *OGAHandler) HandleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		WriteJSONError(w, http.StatusBadRequest, "key is required")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := r.Context()

	// Extract Officer ID from JWT without validating signature (NSW Backend will validate)
	officerID := "unknown"
	if parts := strings.Split(authHeader, " "); len(parts) == 2 {
		tokenParts := strings.Split(parts[1], ".")
		if len(tokenParts) >= 2 {
			if payload, err := base64.RawURLEncoding.DecodeString(tokenParts[1]); err == nil {
				var claims struct {
					Sub string `json:"sub"`
				}
				if json.Unmarshal(payload, &claims) == nil && claims.Sub != "" {
					officerID = claims.Sub
				}
			}
		}
	}

	slog.InfoContext(ctx, "Audit Log: File Access",
		"action", "FILE_DOWNLOAD",
		"officer_id", officerID,
		"key", key,
	)

	// Request a secure download path from the NSW Backend
	backendURL := fmt.Sprintf("%s/api/v1/uploads/%s", h.backendURL, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, backendURL, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create backend request", "error", err)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	// Forward the WSO2 token
	req.Header.Set("Authorization", authHeader)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "backend request failed", "error", err)
		WriteJSONError(w, http.StatusBadGateway, "Backend service unavailable")
		return
	}
	defer resp.Body.Close()
	// Propagate response from backend
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
