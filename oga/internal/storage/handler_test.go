package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockService is a mock implementation of Service for testing
type mockService struct {
	mockCreateUploadURL func(ctx context.Context, payload []byte) (map[string]any, error)
	mockGetDownloadURL  func(ctx context.Context, key string) (map[string]any, error)
}

func (m *mockService) CreateUploadURL(ctx context.Context, payload []byte) (map[string]any, error) {
	if m.mockCreateUploadURL != nil {
		return m.mockCreateUploadURL(ctx, payload)
	}
	return nil, nil
}

func (m *mockService) GetDownloadURL(ctx context.Context, key string) (map[string]any, error) {
	if m.mockGetDownloadURL != nil {
		return m.mockGetDownloadURL(ctx, key)
	}
	return nil, nil
}

func TestHandleCreateUpload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := &mockService{
			mockCreateUploadURL: func(ctx context.Context, payload []byte) (map[string]any, error) {
				return map[string]any{
					"key":        "123-abc",
					"upload_url": "http://test/upload",
				}, nil
			},
		}
		handler := NewHandler(mockSvc, 32<<20)

		body := []byte(`{"filename":"test.txt","mime_type":"text/plain","size":123}`)
		req := httptest.NewRequest(http.MethodPost, "/api/oga/uploads", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		handler.HandleCreateUpload(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response body: %v", err)
		}

		if resp["key"] != "123-abc" {
			t.Errorf("expected key '123-abc', got %v", resp["key"])
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		handler := NewHandler(&mockService{}, 32<<20)
		req := httptest.NewRequest(http.MethodGet, "/api/oga/uploads", nil)
		rec := httptest.NewRecorder()

		handler.HandleCreateUpload(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := &mockService{
			mockCreateUploadURL: func(ctx context.Context, payload []byte) (map[string]any, error) {
				return nil, errors.New("upstream error")
			},
		}
		handler := NewHandler(mockSvc, 32<<20)

		body := []byte(`{"filename":"test.txt","mime_type":"text/plain","size":123}`)
		req := httptest.NewRequest(http.MethodPost, "/api/oga/uploads", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		handler.HandleCreateUpload(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestHandleGetUploadURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := &mockService{
			mockGetDownloadURL: func(ctx context.Context, key string) (map[string]any, error) {
				return map[string]any{
					"download_url": "http://test/download",
					"expires_at":   float64(1234567890),
				}, nil
			},
		}
		handler := NewHandler(mockSvc, 32<<20)

		req := httptest.NewRequest(http.MethodGet, "/api/oga/uploads/550e8400-e29b-41d4-a716-446655440000.pdf", nil)
		req.SetPathValue("key", "550e8400-e29b-41d4-a716-446655440000.pdf") // Set the mux path value
		rec := httptest.NewRecorder()

		handler.HandleGetUploadURL(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response body: %v", err)
		}

		if resp["download_url"] != "http://test/download" {
			t.Errorf("expected download_url 'http://test/download', got %v", resp["download_url"])
		}
		if resp["expires_at"] != float64(1234567890) { // JSON unmarshals ints to float64
			t.Errorf("expected expires_at 1234567890, got %v", resp["expires_at"])
		}
	})
}
