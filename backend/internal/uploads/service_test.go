package uploads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// MockDriver implements StorageDriver for testing
type MockDriver struct {
	SavedKey       string
	SavedBody      []byte
	GenerateURLErr error
	DeleteCalled   bool
	DeleteKey      string
	LastTTL        time.Duration
}

func (m *MockDriver) Save(ctx context.Context, key string, body io.Reader, contentType string) error {
	m.SavedKey = key
	content, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.SavedBody = content
	return nil
}

func (m *MockDriver) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	return io.NopCloser(bytes.NewReader(m.SavedBody)), "application/test", nil
}

func (m *MockDriver) Delete(ctx context.Context, key string) error {
	m.DeleteCalled = true
	m.DeleteKey = key
	return nil
}

func (m *MockDriver) GetDownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	m.LastTTL = ttl
	if m.GenerateURLErr != nil {
		return "", m.GenerateURLErr
	}
	return "/test/download/" + key, nil
}

func TestUploadService(t *testing.T) {
	mock := &MockDriver{}
	service := NewUploadService(mock)

	ctx := context.Background()
	filename := "test.jpg"
	content := []byte("image data")

	metadata, err := service.Upload(ctx, filename, bytes.NewReader(content), int64(len(content)), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if metadata.Name != filename {
		t.Errorf("expected name %s, got %s", filename, metadata.Name)
	}

	if !bytes.Equal(mock.SavedBody, content) {
		t.Error("saved body does not match input")
	}

	if metadata.URL != "" {
		t.Errorf("expected URL to be empty, got %s", metadata.URL)
	}
}

func TestUploadService_Download(t *testing.T) {
	mock := &MockDriver{
		SavedBody: []byte("test content"),
	}
	service := NewUploadService(mock)

	ctx := context.Background()
	reader, contentType, err := service.Download(ctx, "test-key")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	if contentType != "application/test" {
		t.Errorf("expected content type application/test, got %s", contentType)
	}

	content, _ := io.ReadAll(reader)
	if !bytes.Equal(content, mock.SavedBody) {
		t.Error("downloaded content does not match saved body")
	}
}

func TestUploadService_GetDownloadURL_Success(t *testing.T) {
	mock := &MockDriver{}
	service := NewUploadService(mock)

	ctx := context.Background()
	const key = "test-key"

	ttl := 10 * time.Minute
	url, err := service.GetDownloadURL(ctx, key, ttl)
	if err != nil {
		t.Fatalf("GetDownloadURL failed: %v", err)
	}

	if url != "/test/download/"+key {
		t.Errorf("unexpected URL: %s", url)
	}
	if mock.LastTTL != ttl {
		t.Errorf("expected TTL %v, got %v", ttl, mock.LastTTL)
	}
}

func TestUploadService_GetDownloadURL_Error(t *testing.T) {
	expectedErr := io.ErrUnexpectedEOF
	mock := &MockDriver{GenerateURLErr: expectedErr}
	service := NewUploadService(mock)

	_, err := service.GetDownloadURL(context.Background(), "test-key", 0)
	if err == nil {
		t.Fatal("expected error from GetDownloadURL, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}
