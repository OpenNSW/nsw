package model

import "encoding/json"

type TemplateType string

const (
	TemplateTypeForm     TemplateType = "FORM"
	TemplateTypeMarkdown TemplateType = "MARKDOWN"
)

type TemplateSection struct {
	Type    TemplateType    `json:"type"`
	Content json.RawMessage `json:"content"`
}

type TemplateDefinition struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	ROView        TemplateSection `json:"roView"`
	OfficerAction TemplateSection `json:"officerInput"`
}
