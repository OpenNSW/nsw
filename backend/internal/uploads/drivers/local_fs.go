package drivers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LocalFSDriver implements StorageDriver for local disk with directory hashing
type LocalFSDriver struct {
	BaseDir   string
	PublicURL string
}

// NewLocalFSDriver creates a new LocalFSDriver.
// baseDir is where files will be stored.
// publicURL is the base URL used to generate public links (e.g., /api/uploads).
func NewLocalFSDriver(baseDir, publicURL string) (*LocalFSDriver, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &LocalFSDriver{BaseDir: baseDir, PublicURL: publicURL}, nil
}

// getHashedPath generates a two-level deep path for a key to avoid flat directory issues.
func (d *LocalFSDriver) getHashedPath(key string) string {
	if len(key) < 4 {
		return key
	}
	return filepath.Join(key[0:2], key[2:4], key)
}

func (d *LocalFSDriver) Save(ctx context.Context, key string, body io.Reader, contentType string) error {
	fullPath := filepath.Join(d.BaseDir, d.getHashedPath(key))

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create hashed directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, body); err != nil {
		file.Close()
		os.Remove(fullPath)
		return fmt.Errorf("failed to save file content: %w", err)
	}

	// Save metadata sidecar
	metaPath := fullPath + ".meta"
	if err := os.WriteFile(metaPath, []byte(contentType), 0644); err != nil {
		// Try to cleanup content file if metadata save fails
		os.Remove(fullPath)
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

func (d *LocalFSDriver) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	fullPath := filepath.Join(d.BaseDir, d.getHashedPath(key))
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, "", err
	}

	// Try to read metadata sidecar
	metaPath := fullPath + ".meta"
	contentType := "application/octet-stream"
	if metaBytes, err := os.ReadFile(metaPath); err == nil {
		contentType = string(metaBytes)
	}

	return f, contentType, nil
}

func (d *LocalFSDriver) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(d.BaseDir, d.getHashedPath(key))
	os.Remove(fullPath + ".meta") // Ignore error if meta doesn't exist
	err := os.Remove(fullPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (d *LocalFSDriver) GenerateURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	// For local storage, we return a URL relative to our API or a file path if configured.
	// We assume the router will handle /uploads/{key} logic.
	if d.PublicURL == "" {
		return key, nil
	}
	return fmt.Sprintf("%s/%s", d.PublicURL, key), nil
}
