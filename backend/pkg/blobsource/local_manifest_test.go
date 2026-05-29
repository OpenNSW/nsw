package blobsource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeManifest writes a manifest.json at <dir>/manifest.json with the given byId map.
func writeManifest(t *testing.T, dir string, byID map[string]string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"byId": byID})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), body, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestLocalManifest_LoadsViaManifest(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	writeBlobFile(t, nested, "first.json", `{"schema":{"type":"object"}}`)
	writeBlobFile(t, nested, "second.json", `{"schema":{"type":"string"}}`)
	writeManifest(t, dir, map[string]string{
		"my-first":  "nested/first.json",
		"my-second": "nested/second.json",
	})

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	if _, ok, err := src.Get(context.Background(), "my-first"); err != nil || !ok {
		t.Errorf("expected my-first to be loaded (ok=%v, err=%v)", ok, err)
	}
	if _, ok, err := src.Get(context.Background(), "my-second"); err != nil || !ok {
		t.Errorf("expected my-second to be loaded (ok=%v, err=%v)", ok, err)
	}
	// Files that exist on disk but aren't in the manifest must NOT be served by ID.
	if _, ok, _ := src.Get(context.Background(), "first"); ok {
		t.Errorf("flat-mode lookup must not work in manifest mode")
	}
}

func TestLocalManifest_RejectsParentTraversal(t *testing.T) {
	dir := t.TempDir()
	writeBlobFile(t, dir, "stay.json", `{}`)
	writeManifest(t, dir, map[string]string{"escape": "../escape.json"})

	_, err := NewLocal(dir)
	if err == nil {
		t.Fatal("expected error for manifest path escaping the manifest dir")
	}
	if !strings.Contains(err.Error(), "escape") {
		t.Errorf("expected error to mention escape, got %v", err)
	}
}

func TestLocalManifest_RejectsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, map[string]string{"abs": "/etc/hostname"})

	if _, err := NewLocal(dir); err == nil {
		t.Fatal("expected error for absolute manifest path")
	}
}

func TestLocalManifest_ErrorOnMissingBlobFile(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, map[string]string{"alpha": "missing.json"})

	if _, err := NewLocal(dir); err == nil {
		t.Fatal("expected error when manifest references missing file")
	}
}

func TestLocalManifest_RequiresByIDField(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"),
		[]byte(`{"workflows":[]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, err := NewLocal(dir); err == nil {
		t.Fatal("expected error when manifest lacks byId field")
	}
}

func TestLocalManifest_InvalidJSONFailsFast(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"),
		[]byte(`{not valid json`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, err := NewLocal(dir); err == nil {
		t.Fatal("expected error for invalid manifest JSON")
	}
}
