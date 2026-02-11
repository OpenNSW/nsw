package uploads

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/google/uuid"
)

// UploadService coordinates file uploads and manages metadata
type UploadService struct {
	Driver StorageDriver
}

func NewUploadService(driver StorageDriver) *UploadService {
	return &UploadService{Driver: driver}
}

// Upload handles the incoming file, saves it via driver, and returns metadata
func (s *UploadService) Upload(ctx context.Context, filename string, reader io.Reader, size int64, mime string) (*FileMetadata, error) {
	if mime == "" {
		mime = "application/octet-stream"
	}
	id := uuid.New()
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("%s%s", id.String(), ext)

	err := s.Driver.Save(ctx, key, reader, mime)
	if err != nil {
		return nil, fmt.Errorf("storage driver failed: %w", err)
	}

	url, err := s.Driver.GenerateURL(ctx, key, 0)
	if err != nil {
		if delErr := s.Driver.Delete(ctx, key); delErr != nil {
			slog.WarnContext(ctx, "failed to cleanup orphaned file", "key", key, "error", delErr)
		}
		return nil, fmt.Errorf("failed to generate URL: %w", err)
	}

	metadata := &FileMetadata{
		ID:       id,
		Name:     filename,
		Key:      key,
		URL:      url,
		Size:     size,
		MimeType: mime,
	}

	slog.InfoContext(ctx, "File uploaded successfully", "id", id, "key", key)
	return metadata, nil
}

// Download retrieves the file content and its MIME type
func (s *UploadService) Download(ctx context.Context, key string) (io.ReadCloser, string, error) {
	return s.Driver.Get(ctx, key)
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
