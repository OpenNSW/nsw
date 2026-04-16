package uploads

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// UploadService coordinates file uploads and manages metadata
type UploadService struct {
	Driver StorageDriver
}

func NewUploadService(driver StorageDriver) *UploadService {
	return &UploadService{Driver: driver}
}

// Upload handles the preparation of a file upload by generating a unique key
// and a presigned/upload URL via the storage driver.
func (s *UploadService) Upload(ctx context.Context, filename string, size int64, mime string) (*FileMetadata, error) {
	if mime == "" {
		mime = "application/octet-stream"
	}
	id := uuid.NewString()
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("%s%s", id, ext)

	// Generate a presigned URL for the upload.
	// We use a 15-minute TTL by default.
	uploadURL, err := s.Driver.GetUploadURL(ctx, key, 15*time.Minute, mime, size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	metadata := &FileMetadata{
		ID:        id,
		Name:      filename,
		Key:       key,
		UploadURL: uploadURL,
		Size:      size,
		MimeType:  mime,
	}

	slog.InfoContext(ctx, "File upload prepared", "id", id, "key", key)
	return metadata, nil
}

// Download retrieves the file content and its MIME type
func (s *UploadService) Download(ctx context.Context, key string) (io.ReadCloser, string, error) {
	return s.Driver.Get(ctx, key)
}

// GetDownloadURL generates a time-limited or presigned URL for the given key
func (s *UploadService) GetDownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return s.Driver.GetDownloadURL(ctx, key, ttl)
}

// Delete removes a file from storage
func (s *UploadService) Delete(ctx context.Context, key string) error {
	err := s.Driver.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	slog.InfoContext(ctx, "File deleted successfully", "key", key)
	return nil
}
