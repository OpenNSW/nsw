package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/oga/pkg/httpclient"
)

func TestService_CreateUploadURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/uploads", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"key":"123-abc", "upload_url":"http://test/upload"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := httpclient.NewClientBuilder().
		WithBaseURL(server.URL + "/").
		Build()

	service := NewOGAService(nil, nil, client)

	payload := []byte(`{"filename":"test.txt"}`)
	ctx := context.Background()

	result, err := service.CreateUploadURL(ctx, payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result["key"] != "123-abc" {
		t.Errorf("expected key '123-abc', got %v", result["key"])
	}
	if result["upload_url"] != "http://test/upload" {
		t.Errorf("expected upload_url 'http://test/upload', got %v", result["upload_url"])
	}
}

func TestService_GetDownloadURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/uploads/550e8400-e29b-41d4-a716-446655440000.pdf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"download_url":"http://test/download"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := httpclient.NewClientBuilder().
		WithBaseURL(server.URL + "/").
		Build()

	service := NewOGAService(nil, nil, client)
	ctx := context.Background()

	url, err := service.GetDownloadURL(ctx, "550e8400-e29b-41d4-a716-446655440000.pdf")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if url != "http://test/download" {
		t.Errorf("expected download_url 'http://test/download', got %v", url)
	}
}
