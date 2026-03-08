package uploads

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/OpenNSW/nsw/internal/auth"
)

// validStorageKey returns true if key matches UUID or UUID plus extension (e.g. .pdf).
var storageKeyRx = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}(\.[a-zA-Z0-9]+)?$`)

func validStorageKey(key string) bool {
	return len(key) >= 36 && storageKeyRx.MatchString(key)
}

type HTTPHandler struct {
	Service *UploadService
}

func NewHTTPHandler(service *UploadService) *HTTPHandler {
	return &HTTPHandler{Service: service}
}

// writeJSONError sets Content-Type: application/json and writes a consistent JSON error body.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *HTTPHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if auth.GetAuthContext(r.Context()) == nil {
		slog.WarnContext(r.Context(), "authentication required but not provided for upload")
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse form or request too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	metadata, err := h.Service.Upload(r.Context(), header.Filename, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		slog.ErrorContext(r.Context(), "Upload failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode response", "error", err)
	}
}

func (h *HTTPHandler) Download(w http.ResponseWriter, r *http.Request) {
	if auth.GetAuthContext(r.Context()) == nil {
		slog.WarnContext(r.Context(), "authentication required but not provided for download")
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		writeJSONError(w, http.StatusBadRequest, "key is required")
		return
	}
	if !validStorageKey(key) {
		writeJSONError(w, http.StatusBadRequest, "invalid key format")
		return
	}

	url, err := h.Service.GetDownloadURL(r.Context(), key, 15*time.Minute)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to generate download URL", "key", key, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to generate access")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"download_url": url,
		"expires_at":   time.Now().Add(15 * time.Minute).Unix(),
	}); err != nil {
		slog.ErrorContext(r.Context(), "Failed to encode response", "error", err)
	}
}

// DownloadContent streams the file body with auth. Use for "View" in frontend so the
// client can fetch with auth and open a blob URL in a new tab (avoids relative download_url
// opening on the frontend origin and redirecting to login).
func (h *HTTPHandler) DownloadContent(w http.ResponseWriter, r *http.Request) {
	if auth.GetAuthContext(r.Context()) == nil {
		slog.WarnContext(r.Context(), "authentication required but not provided for download content")
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		writeJSONError(w, http.StatusBadRequest, "key is required")
		return
	}
	if !validStorageKey(key) {
		writeJSONError(w, http.StatusBadRequest, "invalid key format")
		return
	}

	body, contentType, err := h.Service.Download(r.Context(), key)
	if err != nil {
		slog.ErrorContext(r.Context(), "Download content failed", "key", key, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get file")
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline")
	_, err = io.Copy(w, body)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to stream download content", "key", key, "error", err)
	}
}

func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if auth.GetAuthContext(r.Context()) == nil {
		slog.WarnContext(r.Context(), "authentication required but not provided for delete")
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	key := r.PathValue("key")
	if key == "" {
		writeJSONError(w, http.StatusBadRequest, "key is required")
		return
	}
	if !validStorageKey(key) {
		writeJSONError(w, http.StatusBadRequest, "invalid key format")
		return
	}

	if err := h.Service.Delete(r.Context(), key); err != nil {
		slog.ErrorContext(r.Context(), "Delete failed", "error", err, "key", key)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
