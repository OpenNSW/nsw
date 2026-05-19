package templatesource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTemplateFile writes content to <dir>/<name>.
func writeTemplateFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestLocal_LoadsValidTemplates(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "alpha.json", `{"schema":{"type":"object"},"uiSchema":{"type":"VerticalLayout"}}`)
	writeTemplateFile(t, dir, "beta.json", `{"schema":{"type":"object","title":"Beta"}}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if _, ok, err := src.GetTemplate(context.Background(), "alpha"); err != nil || !ok {
		t.Errorf("expected template alpha to be loaded (ok=%v, err=%v)", ok, err)
	}
	if _, ok, err := src.GetTemplate(context.Background(), "beta"); err != nil || !ok {
		t.Errorf("expected template beta to be loaded (ok=%v, err=%v)", ok, err)
	}
}

func TestLocal_GetTemplateReturnsRawJSON(t *testing.T) {
	dir := t.TempDir()
	body := `{"schema":{"type":"object","required":["foo"]},"uiSchema":{"type":"VerticalLayout"}}`
	writeTemplateFile(t, dir, "alpha.json", body)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	raw, ok, err := src.GetTemplate(context.Background(), "alpha")
	if err != nil || !ok {
		t.Fatalf("expected alpha to be loaded (ok=%v, err=%v)", ok, err)
	}

	// Verify the returned bytes round-trip through JSON unmarshal.
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("returned template is not valid JSON: %v", err)
	}
	if _, ok := got["schema"]; !ok {
		t.Errorf("expected schema field in returned template, got %v", got)
	}
}

func TestLocal_SkipsNonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "alpha.json", `{"schema":{"type":"object"}}`)
	writeTemplateFile(t, dir, "readme.txt", `this is not a template`)
	writeTemplateFile(t, dir, "beta.yaml", `schema: {}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if _, ok, _ := src.GetTemplate(context.Background(), "alpha"); !ok {
		t.Errorf("expected alpha to be loaded")
	}
	// IDs should be derived from .json filenames only, never from .txt/.yaml.
	if _, ok, _ := src.GetTemplate(context.Background(), "readme"); ok {
		t.Errorf("readme.txt should have been skipped")
	}
	if _, ok, _ := src.GetTemplate(context.Background(), "beta"); ok {
		t.Errorf("beta.yaml should have been skipped")
	}
}

func TestLocal_GetTemplateMiss(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "alpha.json", `{"schema":{"type":"object"}}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	_, ok, err := src.GetTemplate(context.Background(), "does-not-exist")
	if ok {
		t.Errorf("expected GetTemplate miss to return ok=false")
	}
	if err != nil {
		t.Errorf("expected GetTemplate miss to return nil error, got %v", err)
	}
}

func TestLocal_ErrorOnInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "broken.json", `{this is not valid json`)

	_, err := NewLocal(dir)
	if err == nil {
		t.Fatalf("expected error when loading invalid JSON, got nil")
	}
}

func TestLocal_ErrorOnMissingDir(t *testing.T) {
	root := t.TempDir()
	_, err := NewLocal(filepath.Join(root, "does-not-exist"))
	if err == nil {
		t.Fatalf("expected error when templates directory is missing, got nil")
	}
}

func TestLocal_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()
	// A nested directory under dir should be ignored, not recursed into.
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	writeTemplateFile(t, dir, "nested/should_be_ignored.json", `{"schema":{}}`)
	writeTemplateFile(t, dir, "top.json", `{"schema":{"type":"object"}}`)

	src, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if _, ok, _ := src.GetTemplate(context.Background(), "top"); !ok {
		t.Errorf("expected top to be loaded")
	}
	if _, ok, _ := src.GetTemplate(context.Background(), "should_be_ignored"); ok {
		t.Errorf("nested file should not be discovered")
	}
	if _, ok, _ := src.GetTemplate(context.Background(), "nested/should_be_ignored"); ok {
		t.Errorf("nested file should not be discovered under any key")
	}
}
