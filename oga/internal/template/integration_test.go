package template

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTemplateRegistry_Integration_LoadAndUseTemplateDefinition(t *testing.T) {
	dir := t.TempDir()
	templateID := "npqs:phytosanitary:v1"

	mockTemplate := TemplateDefinition{
		ID:         templateID,
		Name:       "NPQS Phytosanitary Review",
		ActionType: ActionTypeReview,
		View: TemplateSection{
			Type: TemplateTypeForm,
			Content: mustMarshalJSON(map[string]any{
				"schema": map[string]any{
					"type":  "object",
					"title": "View Schema",
					"properties": map[string]any{
						"consignmentNo": map[string]any{"type": "string"},
					},
				},
				"uiSchema": map[string]any{
					"type": "VerticalLayout",
					"elements": []any{
						map[string]any{"type": "Control", "scope": "#/properties/consignmentNo"},
					},
				},
			}),
		},
		Action: TemplateSection{
			Type: TemplateTypeForm,
			Content: mustMarshalJSON(map[string]any{
				"schema": map[string]any{
					"type":     "object",
					"title":    "Action Schema",
					"required": []any{"decision"},
					"properties": map[string]any{
						"decision": map[string]any{
							"type": "string",
							"oneOf": []any{
								map[string]any{"const": "APPROVED", "title": "Approved"},
								map[string]any{"const": "REJECTED", "title": "Rejected"},
							},
						},
					},
				},
				"uiSchema": map[string]any{
					"type": "VerticalLayout",
					"elements": []any{
						map[string]any{"type": "Control", "scope": "#/properties/decision"},
					},
				},
			}),
		},
	}

	writeTemplateFile(t, dir, templateID, mockTemplate)

	registry, err := NewTemplateRegistry(dir, templateID)
	if err != nil {
		t.Fatalf("NewTemplateRegistry() error = %v", err)
	}

	tpl, err := registry.GetDefaultTemplate()
	if err != nil {
		t.Fatalf("GetDefaultTemplate() error = %v", err)
	}

	if tpl.ID != templateID {
		t.Fatalf("expected template ID %q, got %q", templateID, tpl.ID)
	}
	if tpl.Name != "NPQS Phytosanitary Review" {
		t.Fatalf("unexpected template name: %q", tpl.Name)
	}
	if tpl.ActionType != ActionTypeReview {
		t.Fatalf("expected ActionType %q, got %q", ActionTypeReview, tpl.ActionType)
	}
	if tpl.View.Type != TemplateTypeForm || tpl.Action.Type != TemplateTypeForm {
		t.Fatalf("expected both sections to be FORM, got view=%q action=%q", tpl.View.Type, tpl.Action.Type)
	}

	viewContent := decodeSectionContent(t, tpl.View.Content)
	actionContent := decodeSectionContent(t, tpl.Action.Content)
	if schemaTitle(viewContent) != "View Schema" {
		t.Fatalf("expected view schema title %q, got %q", "View Schema", schemaTitle(viewContent))
	}
	if schemaTitle(actionContent) != "Action Schema" {
		t.Fatalf("expected action schema title %q, got %q", "Action Schema", schemaTitle(actionContent))
	}

	processed, err := NewFormHandler().Process(context.Background(), tpl.Action.Content, map[string]any{"decision": "APPROVED"})
	if err != nil {
		t.Fatalf("FormHandler.Process() error = %v", err)
	}
	processedRaw, ok := processed.(json.RawMessage)
	if !ok {
		t.Fatalf("expected processed value to be json.RawMessage, got %T", processed)
	}
	if string(processedRaw) != string(tpl.Action.Content) {
		t.Fatalf("expected processed content to match action content")
	}
}

func decodeSectionContent(t *testing.T, content json.RawMessage) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("failed to decode section content: %v", err)
	}
	return decoded
}

func schemaTitle(section map[string]any) string {
	schema, ok := section["schema"].(map[string]any)
	if !ok {
		return ""
	}
	title, _ := schema["title"].(string)
	return title
}
