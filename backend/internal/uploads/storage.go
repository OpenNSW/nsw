package uploads

import (
	"context"
	"io"
	"time"
)

// StorageDriver defines how we interact with the binary storage
type StorageDriver interface {
	// Save writes the content to the storage and returns a unique identifier (key/path)
	Save(ctx context.Context, key string, body io.Reader, contentType string) error

	// Get returns a ReadCloser to stream the file back and its content type
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)

	// Delete removes the file
	Delete(ctx context.Context, key string) error

	// GenerateURL returns a public-facing URL
	GenerateURL(ctx context.Context, key string, expires time.Duration) (string, error)
}
