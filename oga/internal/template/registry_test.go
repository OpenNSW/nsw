package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTemplateRegistry_LoadsTemplatesAndDefault(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "default", buildTemplateDefinition("default", TemplateTypeForm, minimalFormContent(), TemplateTypeForm, minimalFormContent()))
	writeTemplateFile(t, dir, "custom", buildTemplateDefinition("custom", TemplateTypeForm, minimalFormContent(), TemplateTypeForm, minimalFormContent()))

	registry, err := NewTemplateRegistry(dir, "default")
	if err != nil {
		t.Fatalf("NewTemplateRegistry() error = %v", err)
	}

	if _, err := registry.GetTemplate("custom"); err != nil {
		t.Fatalf("GetTemplate(custom) error = %v", err)
	}

	if _, err := registry.GetDefaultTemplate(); err != nil {
		t.Fatalf("GetDefaultTemplate() error = %v", err)
	}
}

func TestNewTemplateRegistry_FailsForInvalidFormContent(t *testing.T) {
	dir := t.TempDir()
	invalidContent := map[string]any{
		"schema": map[string]any{"type": "object"},
	}
	writeTemplateFile(t, dir, "default", buildTemplateDefinition("default", TemplateTypeForm, invalidContent, TemplateTypeForm, minimalFormContent()))

	_, err := NewTemplateRegistry(dir, "default")
	if err == nil {
		t.Fatalf("expected error for invalid form content")
	}
	if !strings.Contains(err.Error(), "uiSchema is required") {
		t.Fatalf("expected uiSchema validation error, got %v", err)
	}
}

func TestNewTemplateRegistry_FailsForUnsupportedSectionType(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "default", buildTemplateDefinition("default", TemplateTypeMarkdown, map[string]any{"text": "hello"}, TemplateTypeForm, minimalFormContent()))

	_, err := NewTemplateRegistry(dir, "default")
	if err == nil {
		t.Fatalf("expected error for unsupported template section type")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestNewTemplateRegistry_FailsWhenDefaultTemplateMissing(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "custom", buildTemplateDefinition("custom", TemplateTypeForm, minimalFormContent(), TemplateTypeForm, minimalFormContent()))

	_, err := NewTemplateRegistry(dir, "default")
	if err == nil {
		t.Fatalf("expected error when default template is missing")
	}
	if !strings.Contains(err.Error(), "default template") {
		t.Fatalf("expected default template error, got %v", err)
	}
}

func TestNewTemplateRegistry_FailsWhenTemplateIDMismatchesFilename(t *testing.T) {
	dir := t.TempDir()
	writeTemplateFile(t, dir, "default", buildTemplateDefinition("mismatch-id", TemplateTypeForm, minimalFormContent(), TemplateTypeForm, minimalFormContent()))

	_, err := NewTemplateRegistry(dir, "default")
	if err == nil {
		t.Fatalf("expected error when template ID mismatches filename")
	}
	if !strings.Contains(err.Error(), "does not match filename") {
		t.Fatalf("expected filename mismatch error, got %v", err)
	}
}

func buildTemplateDefinition(id string, roType TemplateType, roContent map[string]any, officerType TemplateType, officerContent map[string]any) TemplateDefinition {
	return TemplateDefinition{
		ID:         id,
		Name:       id,
		ActionType: ActionTypeReview,
		View: TemplateSection{
			Type:    roType,
			Content: mustMarshalJSON(roContent),
		},
		Action: TemplateSection{
			Type:    officerType,
			Content: mustMarshalJSON(officerContent),
		},
	}
}

func minimalFormContent() map[string]any {
	return map[string]any{
		"schema": map[string]any{
			"type": "object",
		},
		"uiSchema": map[string]any{
			"type":     "VerticalLayout",
			"elements": []any{},
		},
	}
}

func writeTemplateFile(t *testing.T, dir string, id string, definition TemplateDefinition) {
	t.Helper()

	data, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("failed to marshal template definition: %v", err)
	}

	path := filepath.Join(dir, id+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write template file %q: %v", path, err)
	}
}

func mustMarshalJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
