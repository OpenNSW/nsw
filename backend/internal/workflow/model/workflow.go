package model

import (
	"encoding/json"
)

// StepType represents the type of step within a workflow.
type StepType string

const (
	StepTypeSimpleForm   StepType = "SIMPLE_FORM"    // Step for simple form submission
	StepTypeWaitForEvent StepType = "WAIT_FOR_EVENT" // Step that waits for an external event to occur
)

// Node represents an individual step within a workflow template.
type Node struct {
	NodeID    string          `json:"nodeId"`    // Unique identifier for the step
	Type      StepType        `json:"type"`      // Type of the step
	Config    json.RawMessage `json:"config"`    // Configuration specific to the step type
	DependsOn []string        `json:"dependsOn"` // List of step IDs that this step depends on
}

// HSCodeWorkflow represents the template of a workflow for a specific HS code.
type HSCodeWorkflow struct {
	BaseModel
	Version string `gorm:"type:varchar(50);column:version;not null" json:"version"`       // Version of the workflow template
	Nodes   []Node `gorm:"type:jsonb;column:steps;serializer:json;not null" json:"steps"` // List of steps in the workflow template
}

func (w *HSCodeWorkflow) TableName() string {
	return "workflow_templates"
}

// WorkflowTemplate represents the template of a workflow for consignments.
type WorkflowTemplate = HSCodeWorkflow
