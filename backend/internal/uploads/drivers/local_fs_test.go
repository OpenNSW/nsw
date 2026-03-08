package drivers

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFSDriver_DirectoryHashing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "localfs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewLocalFSDriver(tempDir, "/uploads")
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	key := "abcdef123456.pdf"
	content := []byte("test content")

	// Test Save
	err = driver.Save(ctx, key, bytes.NewReader(content), "application/pdf")
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}

	// Verify Hashing: key "abcdef123456.pdf" should be at ab/cd/abcdef123456.pdf
	expectedSubPath := filepath.Join("ab", "cd", key)
	fullPath := filepath.Join(tempDir, expectedSubPath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("file not found at hashed path: %s", fullPath)
	}

	// Test Get
	reader, contentType, err := driver.Get(ctx, key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	defer reader.Close()

	if contentType != "application/pdf" {
		t.Errorf("expected content type application/pdf, got %s", contentType)
	}

	// Verify URL
	url, err := driver.GenerateURL(ctx, key, 0)
	if err != nil {
		t.Errorf("GenerateURL failed: %v", err)
	}
	if !strings.HasSuffix(url, key) || !strings.Contains(url, "/uploads") {
		t.Errorf("unexpected URL: %s", url)
	}

	// Test Delete
	err = driver.Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file still exists after deletion")
	}
}

func TestLocalFSDriver_RejectsPathTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "localfs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewLocalFSDriver(tempDir, "/uploads")
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	ctx := context.Background()
	badKeys := []string{"../../../etc/passwd", "ab/cd/../../secret", "key\\with\\backslash"}

	for _, key := range badKeys {
		err = driver.Save(ctx, key, bytes.NewReader([]byte("x")), "text/plain")
		if err == nil {
			t.Errorf("Save with key %q should have failed", key)
		}
		_, _, err = driver.Get(ctx, key)
		if err == nil {
			t.Errorf("Get with key %q should have failed", key)
		}
		err = driver.Delete(ctx, key)
		if err == nil {
			t.Errorf("Delete with key %q should have failed", key)
		}
	}
}
