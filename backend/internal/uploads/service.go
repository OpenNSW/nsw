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

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

type countingReadSeeker struct {
	*countingReader
	s io.Seeker
}

func (c *countingReadSeeker) Seek(offset int64, whence int) (int64, error) {
	pos, err := c.s.Seek(offset, whence)
	if err == nil && pos == 0 {
		c.n = 0
	}
	return pos, err
}

// UploadLegacy handles the incoming file, saves it via driver directly, and returns metadata.
// Serves as a bridge for existing UI clients using multipart/form-data.
func (s *UploadService) UploadLegacy(ctx context.Context, filename string, reader io.Reader, size int64, mime string) (*FileMetadata, error) {
	if mime == "" {
		mime = "application/octet-stream"
	}
	id := uuid.NewString()
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("%s%s", id, ext)

	cr := &countingReader{r: reader}
	var body io.Reader = cr
	if seeker, ok := reader.(io.Seeker); ok {
		body = &countingReadSeeker{countingReader: cr, s: seeker}
	}

	err := s.Driver.Save(ctx, key, body, mime)
	if err != nil {
		return nil, fmt.Errorf("storage driver failed: %w", err)
	}

	metadata := &FileMetadata{
		ID:       id,
		Name:     filename,
		Key:      key,
		URL:      "",
		Size:     cr.n,
		MimeType: mime,
	}

	slog.InfoContext(ctx, "File uploaded successfully (legacy)", "id", id, "key", key)
	return metadata, nil
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
