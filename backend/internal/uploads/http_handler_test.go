package uploads

import (
	"context"
	"encoding/json"
	"errors"
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

	req := httptest.NewRequest(http.MethodGet, "/files/my-file-key", nil)
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
	if url != "/test/download/my-file-key" {
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

	req := httptest.NewRequest(http.MethodGet, "/files/bad-key", nil)
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
