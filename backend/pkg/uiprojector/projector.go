package uiprojector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
)

// ProjectorType identifies a projector implementation. External packages may
// declare their own ProjectorType constants to register custom projectors.
type ProjectorType string

// Projector defines the interface for transforming raw template + data into a UI payload.
// Type returns the identifier under which the projector is registered; it must match
// the SectionBlueprint.Projector values used in blueprints.
type Projector interface {
	Type() ProjectorType
	Project(ctx context.Context, templateContent []byte, data any) (any, error)
}

// FormProjector projects raw JSON schema into a FormContent payload.
type FormProjector struct{}

func NewFormProjector() *FormProjector {
	return &FormProjector{}
}

func (p *FormProjector) Type() ProjectorType { return ProjectorForm }

func (p *FormProjector) Project(ctx context.Context, templateContent []byte, data any) (any, error) {
	var schema map[string]any
	if err := json.Unmarshal(templateContent, &schema); err != nil {
		return nil, fmt.Errorf("form_projector: failed to parse schema: %w", err)
	}

	return FormContent{
		Schema:   schema["schema"],
		UISchema: schema["uiSchema"],
		FormData: data,
	}, nil
}

// MarkdownProjector projects a markdown template using Go's text/template.
type MarkdownProjector struct{}

func NewMarkdownProjector() *MarkdownProjector {
	return &MarkdownProjector{}
}

func (p *MarkdownProjector) Type() ProjectorType { return ProjectorMarkdown }

func (p *MarkdownProjector) Project(ctx context.Context, templateContent []byte, data any) (any, error) {
	tmpl, err := template.New("markdown").Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("markdown_projector: failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("markdown_projector: failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RawProjector returns the data as-is without any transformation.
type RawProjector struct{}

func NewRawProjector() *RawProjector {
	return &RawProjector{}
}

func (p *RawProjector) Type() ProjectorType { return ProjectorRaw }

func (p *RawProjector) Project(ctx context.Context, templateContent []byte, data any) (any, error) {
	return data, nil
}
