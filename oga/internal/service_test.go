package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGetUploadURL_WithAuthToken tests that when an auth token is configured,
// the service calls the backend's authenticated endpoint and returns the presigned URL.
func TestGetUploadURL_WithAuthToken(t *testing.T) {
	// Create a mock backend server that returns a presigned URL
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is authenticated
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token-123" {
			t.Errorf("expected Authorization header 'Bearer test-token-123', got %q", authHeader)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify the correct endpoint is called
		expectedPath := "/api/v1/uploads/test-key-uuid.pdf"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		// Return a mock presigned URL response
		response := map[string]any{
			"download_url": "https://s3.amazonaws.com/bucket/test-key-uuid.pdf?presigned-params",
			"expires_at":   time.Now().Add(15 * time.Minute).Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockBackend.Close()

	// Create a service with an auth token
	service := &ogaService{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		nswAPIBaseURL: mockBackend.URL + "/api/v1",
		authToken:     "test-token-123",
	}

	// Call GetUploadURL
	ctx := context.Background()
	downloadURL, err := service.GetUploadURL(ctx, "test-key-uuid.pdf", mockBackend.URL+"/api/v1")
	if err != nil {
		t.Fatalf("GetUploadURL failed: %v", err)
	}

	// Verify the returned URL is the presigned URL from the backend
	expectedURL := "https://s3.amazonaws.com/bucket/test-key-uuid.pdf?presigned-params"
	if downloadURL != expectedURL {
		t.Errorf("expected download URL %q, got %q", expectedURL, downloadURL)
	}
}

// TestGetUploadURL_WithoutAuthToken tests that when no auth token is configured,
// the service falls back to constructing a direct /content URL (for LocalFS).
func TestGetUploadURL_WithoutAuthToken(t *testing.T) {
	// Create a service without an auth token
	service := &ogaService{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		nswAPIBaseURL: "http://localhost:8080/api/v1",
		authToken:     "", // No auth token
	}

	// Call GetUploadURL
	ctx := context.Background()
	downloadURL, err := service.GetUploadURL(ctx, "test-key-uuid.pdf", "http://localhost:8080/api/v1")
	if err != nil {
		t.Fatalf("GetUploadURL failed: %v", err)
	}

	// Verify the returned URL is the direct /content URL
	expectedURL := "http://localhost:8080/api/v1/uploads/test-key-uuid.pdf/content"
	if downloadURL != expectedURL {
		t.Errorf("expected download URL %q, got %q", expectedURL, downloadURL)
	}
}

// TestGetUploadURL_BackendError tests error handling when the backend returns an error.
func TestGetUploadURL_BackendError(t *testing.T) {
	// Create a mock backend server that returns an error
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "file not found"}`))
	}))
	defer mockBackend.Close()

	// Create a service with an auth token
	service := &ogaService{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		nswAPIBaseURL: mockBackend.URL + "/api/v1",
		authToken:     "test-token-123",
	}

	// Call GetUploadURL
	ctx := context.Background()
	_, err := service.GetUploadURL(ctx, "nonexistent-key.pdf", mockBackend.URL+"/api/v1")
	if err == nil {
		t.Fatal("expected error when backend returns 404, got nil")
	}

	// Verify the error message contains the status code
	expectedErrSubstring := "backend returned status 404"
	if err.Error()[:len(expectedErrSubstring)] != expectedErrSubstring {
		t.Errorf("expected error to start with %q, got %q", expectedErrSubstring, err.Error())
	}
}

// TestGetUploadURL_BackendUnauthorized tests error handling when auth fails.
func TestGetUploadURL_BackendUnauthorized(t *testing.T) {
	// Create a mock backend server that rejects the auth token
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer mockBackend.Close()

	// Create a service with an invalid auth token
	service := &ogaService{
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		nswAPIBaseURL: mockBackend.URL + "/api/v1",
		authToken:     "invalid-token",
	}

	// Call GetUploadURL
	ctx := context.Background()
	_, err := service.GetUploadURL(ctx, "test-key-uuid.pdf", mockBackend.URL+"/api/v1")
	if err == nil {
		t.Fatal("expected error when backend returns 401, got nil")
	}

	// Verify the error message contains the status code
	expectedErrSubstring := "backend returned status 401"
	if err.Error()[:len(expectedErrSubstring)] != expectedErrSubstring {
		t.Errorf("expected error to start with %q, got %q", expectedErrSubstring, err.Error())
	}
}
