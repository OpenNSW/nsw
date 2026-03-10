package uploads

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/uploads/drivers"
)

// validStorageKey returns true if key matches UUID or UUID plus extension (e.g. .pdf).
var storageKeyRx = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}(\.[a-zA-Z0-9]+)?$`)

func validStorageKey(key string) bool {
	return len(key) >= 36 && storageKeyRx.MatchString(key)
}

var allowedContentTypes = map[string]struct{}{
	"application/pdf": {},
	"image/jpeg":      {},
	"image/png":       {},
	"image/gif":       {},
	"image/webp":      {},
}

func isAllowedContentType(ct string) bool {
	_, ok := allowedContentTypes[ct]
	return ok
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

	// Derive a trustworthy content type from the actual bytes when possible.
	// We avoid trusting client-supplied multipart headers for security and accuracy.
	mimeType := header.Header.Get("Content-Type")
	if seeker, ok := file.(io.ReadSeeker); ok {
		// Sniff content type from the first 512 bytes.
		sniffBuf := make([]byte, 512)
		n, _ := seeker.Read(sniffBuf)
		if n > 0 {
			mimeType = http.DetectContentType(sniffBuf[:n])
		}

		// Rewind so the upload service can read from the beginning. The service
		// is responsible for reporting the actual number of bytes written.
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	if !isAllowedContentType(mimeType) {
		writeJSONError(w, http.StatusUnsupportedMediaType, "invalid or prohibited file type")
		return
	}

	// NOTE: The service layer is responsible for determining the actual number
	// of bytes written; the size passed here is treated only as a hint.
	metadata, err := h.Service.Upload(r.Context(), header.Filename, file, header.Size, mimeType)
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

// DownloadContent streams the file body directly from the local filesystem driver.
// It is intended only for local development when using LocalFSDriver; in non-local
// environments (e.g. S3) callers should use GetDownloadURL and presigned URLs instead.
func (h *HTTPHandler) DownloadContent(w http.ResponseWriter, r *http.Request) {
	// This endpoint is only available when using LocalFSDriver (local development).
	// It serves the same role as an S3 presigned URL — no auth required since the
	// caller was already authenticated when obtaining the URL via GET /uploads/{key}.
	if _, ok := h.Service.Driver.(*drivers.LocalFSDriver); !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
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
	defer func() { _ = body.Close() }()

	w.Header().Set("Content-Type", contentType)
	// Check if the body can report its size (standard for files/drivers)
	w.Header().Set("Content-Disposition", "inline")
	if stater, ok := body.(interface{ Stat() (os.FileInfo, error) }); ok {
		if fi, err := stater.Stat(); err == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		}
	}

	// Ensure headers (including Content-Length) are written before the body so
	// that browsers can correctly display download progress.
	w.WriteHeader(http.StatusOK)

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
