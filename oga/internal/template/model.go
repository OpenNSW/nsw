package template

import "encoding/json"

type TemplateType string

const (
	TemplateTypeForm     TemplateType = "FORM"
	TemplateTypeMarkdown TemplateType = "MARKDOWN"
)

type ActionType string

const (
	ActionTypeReview ActionType = "REVIEW"
	ActionTypeSubmit ActionType = "SUBMIT"
)

type TemplateSection struct {
	Type    TemplateType    `json:"type"`
	Content json.RawMessage `json:"content"`
}

type TemplateDefinition struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	View       TemplateSection `json:"view"`
	Action     TemplateSection `json:"action"`
	ActionType ActionType      `json:"actionType"` // Used by the frontend to determine how to display the action section (e.g. as a review application or submit form)
}
