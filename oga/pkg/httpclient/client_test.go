package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNoAuthenticator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(5*time.Second, nil)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", resp.Status)
	}
}

func TestPost(t *testing.T) {
	expectedBody := "hello world"
	expectedContentType := "text/plain"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %v", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != expectedContentType {
			t.Errorf("expected content type %q, got %q", expectedContentType, ct)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, string(body))
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(5*time.Second, nil)
	resp, err := client.Post(server.URL, expectedContentType, []byte(expectedBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status Created, got %v", resp.Status)
	}
}

// MockAuthenticator is a helper for testing authentication failures.
type MockAuthenticator struct {
	err error
}

func (m *MockAuthenticator) Authenticate(req *http.Request) error {
	return m.err
}

func TestDoAuthenticationFailure(t *testing.T) {
	authErr := fmt.Errorf("auth failed")
	auth := &MockAuthenticator{err: authErr}
	client := NewClient(5*time.Second, auth)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := client.Do(req)

	if err != authErr {
		t.Errorf("expected error %v, got %v", authErr, err)
	}
	if resp != nil {
		t.Error("expected nil response on auth failure")
	}
}

func TestGetInvalidURL(t *testing.T) {
	client := NewClient(5*time.Second, nil)
	_, err := client.Get(":") // Invalid URL
	if err == nil {
		t.Error("expected error for invalid URL in Get")
	}
}

func TestPostInvalidURL(t *testing.T) {
	client := NewClient(5*time.Second, nil)
	_, err := client.Post(":", "text/plain", nil) // Invalid URL
	if err == nil {
		t.Error("expected error for invalid URL in Post")
	}
}
