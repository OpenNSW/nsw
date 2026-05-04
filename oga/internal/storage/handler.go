package storage

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Handler handles HTTP requests for storage operations
type Handler struct {
	service         Service
	MaxRequestBytes int64
}

// NewHandler creates a new storage handler instance
func NewHandler(service Service, maxRequestBytes int64) *Handler {
	return &Handler{
		service:         service,
		MaxRequestBytes: maxRequestBytes,
	}
}

// HandleGetUploadURL returns a download URL for a file stored in the main backend.
func (h *Handler) HandleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeJSONError(w, http.StatusBadRequest, "key is required")
		return
	}
	metadata, err := h.service.GetDownloadURL(r.Context(), key)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get download URL from backend", "key", key, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get download URL")
		return
	}

	writeJSONResponse(w, http.StatusOK, metadata)
}

// HandleCreateUpload prepares an upload by requesting an upload URL from the main backend.
func (h *Handler) HandleCreateUpload(w http.ResponseWriter, r *http.Request) {
	// TODO: Add Authentication & Authorization middleware
	// Access must be restricted to authorized OGA officers to prevent unauthorized users
	// from generating proxy pre-signed upload URLs. Introduce a configuration flag (e.g. OGA_DISABLE_AUTH)
	// to make bypassing explicit for specific environments

	r.Body = http.MaxBytesReader(w, r.Body, h.MaxRequestBytes)
	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.service.CreateUploadURL(r.Context(), body)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create upload URL", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create upload URL")
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// Private helpers to avoid circular dependencies with the main internal package
func writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, map[string]string{
		"error": message,
	})
}
