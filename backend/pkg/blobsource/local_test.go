package blobsource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeBlobFile writes content to <dir>/<name>.
func writeBlobFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestLocal_LoadsBlobs(t *testing.T) {
	dir := t.TempDir()
	writeBlobFile(t, dir, "alpha.json", `{"schema":{"type":"object"},"uiSchema":{"type":"VerticalLayout"}}`)
	writeBlobFile(t, dir, "beta.json", `{"schema":{"type":"object","title":"Beta"}}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if _, ok, err := src.Get(context.Background(), "alpha"); err != nil || !ok {
		t.Errorf("expected blob alpha to be loaded (ok=%v, err=%v)", ok, err)
	}
	if _, ok, err := src.Get(context.Background(), "beta"); err != nil || !ok {
		t.Errorf("expected blob beta to be loaded (ok=%v, err=%v)", ok, err)
	}
}

func TestLocal_GetReturnsRawBytes(t *testing.T) {
	dir := t.TempDir()
	body := `{"schema":{"type":"object","required":["foo"]},"uiSchema":{"type":"VerticalLayout"}}`
	writeBlobFile(t, dir, "alpha.json", body)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	raw, ok, err := src.Get(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected alpha to be loaded (ok=%v, err=%v)", ok, err)
	}

	if string(raw) != body {
		t.Errorf("expected returned bytes to match file content exactly\n got: %s\nwant: %s", raw, body)
	}

	// Sanity check that the test payload itself is still JSON-shaped — proves
	// JSON content still flows through unchanged.
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("returned blob is not valid JSON: %v", err)
	}
	if _, ok := got["schema"]; !ok {
		t.Errorf("expected schema field in returned blob, got %v", got)
	}
}

func TestLocal_SkipsNonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	writeBlobFile(t, dir, "alpha.json", `{"schema":{"type":"object"}}`)
	writeBlobFile(t, dir, "readme.txt", `this is not a blob`)
	writeBlobFile(t, dir, "beta.yaml", `schema: {}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if _, ok, _ := src.Get(context.Background(), "alpha"); !ok {
		t.Errorf("expected alpha to be loaded")
	}
	// IDs should be derived from .json filenames only, never from .txt/.yaml.
	if _, ok, _ := src.Get(context.Background(), "readme"); ok {
		t.Errorf("readme.txt should have been skipped")
	}
	if _, ok, _ := src.Get(context.Background(), "beta"); ok {
		t.Errorf("beta.yaml should have been skipped")
	}
}

func TestLocal_GetMiss(t *testing.T) {
	dir := t.TempDir()
	writeBlobFile(t, dir, "alpha.json", `{"schema":{"type":"object"}}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	_, ok, err := src.Get(context.Background(), "does-not-exist")
	if ok {
		t.Errorf("expected Get miss to return ok=false")
	}
	if err != nil {
		t.Errorf("expected Get miss to return nil error, got %v", err)
	}
}

// TestLocal_LoadsInvalidJSONWithoutError documents that the package no longer
// validates payload syntax — files with invalid JSON load successfully and are
// returned to the caller verbatim.
func TestLocal_LoadsInvalidJSONWithoutError(t *testing.T) {
	dir := t.TempDir()
	body := `{this is not valid json`
	writeBlobFile(t, dir, "broken.json", body)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal should not validate payload syntax; got err: %v", err)
	}
	raw, ok, err := src.Get(context.Background(), "broken")
	if err != nil || !ok {
		t.Fatalf("expected broken blob to be served verbatim (ok=%v, err=%v)", ok, err)
	}
	if string(raw) != body {
		t.Errorf("expected raw bytes unchanged, got %s", raw)
	}
}

func TestLocal_ErrorOnMissingDir(t *testing.T) {
	root := t.TempDir()
	_, err := NewLocal(filepath.Join(root, "does-not-exist"))
	if err == nil {
		t.Fatalf("expected error when blobs directory is missing, got nil")
	}
}

func TestLocal_ErrorOnEmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := NewLocal(dir)
	if err == nil {
		t.Fatalf("expected error for directory with no .json files, got nil")
	}
}

func TestLocal_ErrorOnDirWithNoJSONFiles(t *testing.T) {
	dir := t.TempDir()
	writeBlobFile(t, dir, "readme.txt", "not a blob")
	writeBlobFile(t, dir, "config.yaml", "key: value")

	_, err := NewLocal(dir)
	if err == nil {
		t.Fatalf("expected error when directory contains no .json files, got nil")
	}
}

func TestLocal_Close(t *testing.T) {
	dir := t.TempDir()
	writeBlobFile(t, dir, "alpha.json", `{"schema":{}}`)
	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Errorf("Close returned unexpected error: %v", err)
	}
}

func TestLocal_ErrorOnUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod restrictions do not apply")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "locked.json")
	if err := os.WriteFile(path, []byte(`{"schema":{}}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	_, err := NewLocal(dir)
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestLocal_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()
	// A nested directory under dir should be ignored, not recursed into.
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	writeBlobFile(t, dir, "nested/should_be_ignored.json", `{"schema":{}}`)
	writeBlobFile(t, dir, "top.json", `{"schema":{"type":"object"}}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if _, ok, _ := src.Get(context.Background(), "top"); !ok {
		t.Errorf("expected top to be loaded")
	}
	if _, ok, _ := src.Get(context.Background(), "should_be_ignored"); ok {
		t.Errorf("nested file should not be discovered")
	}
	if _, ok, _ := src.Get(context.Background(), "nested/should_be_ignored"); ok {
		t.Errorf("nested file should not be discovered under any key")
	}
}
