package uploads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/internal/auth"
)

// withAuthContext returns a context with the given AuthContext injected.
func withAuthContext(ctx context.Context, ac *auth.AuthContext) context.Context {
	return context.WithValue(ctx, auth.AuthContextKey, ac)
}

func TestDownload_Unauthorized(t *testing.T) {
	handler := NewHTTPHandler(NewUploadService(&MockDriver{}))

	req := httptest.NewRequest(http.MethodGet, "/files/some-key", nil)
	// No auth context set — should be rejected.
	rec := httptest.NewRecorder()

	handler.Download(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected error body, got empty")
	}
}

func TestDownload_MissingKey(t *testing.T) {
	handler := NewHTTPHandler(NewUploadService(&MockDriver{}))

	req := httptest.NewRequest(http.MethodGet, "/files/", nil)
	// Auth present, but no path value for "key".
	ctx := withAuthContext(req.Context(), &auth.AuthContext{
		TraderContext: &auth.TraderContext{TraderID: "trader-1"},
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Download(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestDownload_Success(t *testing.T) {
	mock := &MockDriver{}
	handler := NewHTTPHandler(NewUploadService(mock))

	// Build request with auth context and path value.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /files/{key}", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/files/550e8400-e29b-41d4-a716-446655440000.pdf", nil)
	ctx := withAuthContext(req.Context(), &auth.AuthContext{
		TraderContext: &auth.TraderContext{TraderID: "trader-1"},
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := resp["download_url"]; !ok {
		t.Error("response missing 'download_url' field")
	}
	if _, ok := resp["expires_at"]; !ok {
		t.Error("response missing 'expires_at' field")
	}

	url, _ := resp["download_url"].(string)
	if url != "/test/download/550e8400-e29b-41d4-a716-446655440000.pdf" {
		t.Errorf("unexpected download_url: %s", url)
	}
}

func TestDownload_GenerateURLError(t *testing.T) {
	mock := &MockDriver{
		GenerateURLErr: errors.New("presign failure"),
	}
	handler := NewHTTPHandler(NewUploadService(mock))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /files/{key}", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/files/550e8400-e29b-41d4-a716-446655440000", nil)
	ctx := withAuthContext(req.Context(), &auth.AuthContext{
		TraderContext: &auth.TraderContext{TraderID: "trader-1"},
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected error body, got empty")
	}
}

func TestDownload_InvalidKeyFormat(t *testing.T) {
	handler := NewHTTPHandler(NewUploadService(&MockDriver{}))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /files/{key}", handler.Download)

	// Key that is not UUID or UUID.ext (validStorageKey rejects it)
	req := httptest.NewRequest(http.MethodGet, "/files/invalid-key-format", nil)
	ctx := withAuthContext(req.Context(), &auth.AuthContext{
		TraderContext: &auth.TraderContext{TraderID: "trader-1"},
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid key, got %d", rec.Code)
	}
}

func TestUpload_Unauthorized(t *testing.T) {
	handler := NewHTTPHandler(NewUploadService(&MockDriver{}))

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.pdf")
	part.Write([]byte("content"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/uploads", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.Upload(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	var errBody map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errBody); err != nil {
		t.Errorf("expected JSON body: %v", err)
	}
	if errBody["error"] == "" {
		t.Error("expected error message in body")
	}
}

func TestDelete_Unauthorized(t *testing.T) {
	handler := NewHTTPHandler(NewUploadService(&MockDriver{}))

	req := httptest.NewRequest(http.MethodDelete, "/uploads/550e8400-e29b-41d4-a716-446655440000.pdf", nil)
	req.SetPathValue("key", "550e8400-e29b-41d4-a716-446655440000.pdf")
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestDownloadContent_NonLocalDriver_NotFound(t *testing.T) {
	// For non-local drivers, DownloadContent should be disabled and return 404
	handler := NewHTTPHandler(NewUploadService(&MockDriver{}))

	req := httptest.NewRequest(http.MethodGet, "/uploads/550e8400-e29b-41d4-a716-446655440000.pdf/content", nil)
	req.SetPathValue("key", "550e8400-e29b-41d4-a716-446655440000.pdf")
	rec := httptest.NewRecorder()

	handler.DownloadContent(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}
