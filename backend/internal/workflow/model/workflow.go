package model

import (
	"encoding/json"
)

// StepType represents the type of a step within a workflow.json.
type StepType string

const (
	StepTypeTraderForm     StepType = "TRADER_FORM"         // Step where trader fills out a form
	StepTypeOGAForm        StepType = "OGA_FORM"            // Step where OGA officer fills out a form
	StepTypeWaitForEvent   StepType = "WAIT_FOR_EVENT"      // Step that waits for an external event to occur
	StepTypeDocumentSubmit StepType = "DOCUMENT_SUBMISSION" // Step for document submission
	StepTypePayment        StepType = "PAYMENT"             // Step for payment processing
)

// Step represents an individual step within a workflow.json template.
type Step struct {
	StepID    string          `json:"stepId"`    // Unique identifier for the step
	Type      StepType        `json:"type"`      // Type of the step
	Config    json.RawMessage `json:"config"`    // Configuration specific to the step type
	DependsOn []string        `json:"dependsOn"` // List of step IDs that this step depends on
}

// WorkflowTemplate represents the template of a workflow.json for consignments.
type WorkflowTemplate struct {
	BaseModel
	Version string `gorm:"type:varchar(50);column:version;not null" json:"version"`       // Version of the workflow.json template
	Steps   []Step `gorm:"type:jsonb;column:steps;serializer:json;not null" json:"steps"` // List of steps in the workflow.json template
}

func (w *WorkflowTemplate) TableName() string {
	return "workflow_templates"
}
