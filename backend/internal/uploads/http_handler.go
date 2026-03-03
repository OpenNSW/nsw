package uploads

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/OpenNSW/nsw/internal/auth"
)

type HTTPHandler struct {
	Service *UploadService
}

func NewHTTPHandler(service *UploadService) *HTTPHandler {
	return &HTTPHandler{Service: service}
}

func (h *HTTPHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Enforce 32MB max request body size to prevent infinite streaming or memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, `{"error": "failed to parse form or request too large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error": "file is required"}`, http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()

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
	// Check Auth via Middleware context
	authCtx := auth.GetAuthContext(r.Context())
	if authCtx == nil {
		slog.WarnContext(r.Context(), "authentication required but not provided for download")
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	key := r.PathValue("key")
	if key == "" {
		http.Error(w, `{"error": "key is required"}`, http.StatusBadRequest)
		return
	}

	// Generate URL with a 15-minute TTL
	url, err := h.Service.GetDownloadURL(r.Context(), key, 15*time.Minute)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to generate download URL", "key", key, "error", err)
		http.Error(w, `{"error": "failed to generate access"}`, http.StatusInternalServerError)
		return
	}

	// Return JSON response with the target URL
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"download_url": url,
		"expires_at":   time.Now().Add(15 * time.Minute).Unix(),
	}); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode response", "error", err)
	}
}

func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, `{"error": "key is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.Service.Delete(r.Context(), key); err != nil {
		slog.ErrorContext(r.Context(), "Delete failed", "error", err, "key", key)
		http.Error(w, `{"error": "failed to delete file"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
