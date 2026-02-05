package uploads

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type HTTPHandler struct {
	Service *UploadService
}

func NewHTTPHandler(service *UploadService) *HTTPHandler {
	return &HTTPHandler{Service: service}
}

func (h *HTTPHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	// Max memory 32MB
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, `{"error": "failed to parse form"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error": "file is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	metadata, err := h.Service.Upload(r.Context(), header.Filename, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		slog.ErrorContext(r.Context(), "Upload failed", "error", err)
		http.Error(w, `{"error": "upload failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode response", "error", err)
	}
}

func (h *HTTPHandler) Download(w http.ResponseWriter, r *http.Request) {
	// Extract key from URL path
	// Assuming URL pattern is /api/uploads/{key} or similar
	// We'll take the last path segment
	parts := strings.Split(r.URL.Path, "/")
	key := parts[len(parts)-1]

	if key == "" {
		http.Error(w, `{"error": "key is required"}`, http.StatusBadRequest)
		return
	}

	reader, contentType, err := h.Service.Download(r.Context(), key)
	if err != nil {
		http.Error(w, `{"error": "file not found"}`, http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	if _, err := io.Copy(w, reader); err != nil {
		slog.ErrorContext(r.Context(), "Failed to copy file content", "error", err)
	}
}
