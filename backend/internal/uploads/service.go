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

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
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

	// Wrap the reader so we can determine the actual number of bytes written,
	// rather than trusting any client-supplied size hints.
	cr := &countingReader{r: reader}

	err := s.Driver.Save(ctx, key, cr, mime)
	if err != nil {
		return nil, fmt.Errorf("storage driver failed: %w", err)
	}

	metadata := &FileMetadata{
		ID:       id,
		Name:     filename,
		Key:      key,
		// URL is not populated by default; clients should call GetDownloadURL
		// when they need a time-limited or presigned URL for download.
		URL:      "",
		Size:     cr.n,
		MimeType: mime,
	}

	slog.InfoContext(ctx, "File uploaded successfully", "id", id, "key", key)
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
