package uploads

import (
	"bytes"
	"context"
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

func (m *MockDriver) GenerateURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if m.GenerateURLErr != nil {
		return "", m.GenerateURLErr
	}
	return "/test/" + key, nil
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

	if metadata.URL != "/test/"+mock.SavedKey {
		t.Errorf("unexpected URL: %s", metadata.URL)
	}
}

func TestUploadService_GenerateURLFailure(t *testing.T) {
	mock := &MockDriver{
		GenerateURLErr: io.ErrUnexpectedEOF, // Just an example error
	}
	service := NewUploadService(mock)

	ctx := context.Background()
	filename := "test_fail.jpg"
	content := []byte("image data")

	_, err := service.Upload(ctx, filename, bytes.NewReader(content), int64(len(content)), "image/jpeg")
	if err == nil {
		t.Fatal("expected Upload to fail when GenerateURL fails")
	}

	if !mock.DeleteCalled {
		t.Error("expected Delete to be called to cleanup orphaned file")
	}

	if mock.DeleteKey != mock.SavedKey {
		t.Errorf("expected Delete to be called with key %s, got %s", mock.SavedKey, mock.DeleteKey)
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
