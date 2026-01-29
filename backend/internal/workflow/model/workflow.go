package model

import (
	"encoding/json"
)

// TaskType represents the type of task within a workflow.
type TaskType string

const (
	TaskTypeSimpleForm   TaskType = "SIMPLE_FORM"    // Task for simple form submission
	TaskTypeWaitForEvent TaskType = "WAIT_FOR_EVENT" // Task that waits for an external event to occur
)

// TaskTemplate represents a static definition of a task process.
// Example: Customs Declaration (CUSDEC)
type TaskTemplate struct {
	ID     string          `json:"id"`     // Unique identifier for the task template (e.g., CUSDEC)
	Name   string          `json:"name"`   // Display name of the task
	Type   TaskType        `json:"type"`   // Type of the task
	Config json.RawMessage `json:"config"` // Configuration specific to the task type
}

// GraphNode represents a node in the dependency graph of an HSCodeWorkflow.
type GraphNode struct {
	NodeID         string   `json:"nodeId"`         // Unique identifier for the node within the workflow (e.g., CusDec) - matches Task.StepID
	NodeTemplateID string   `json:"nodeTemplateId"` // Reference to the TaskTemplate ID (e.g., CUSDEC)
	DependsOn      []string `json:"dependsOn"`      // List of NodeIDs that this node depends on
}

// HSCodeWorkflow represents the template of a workflow for a specific HS Code.
type HSCodeWorkflow struct {
	BaseModel
	Version string      `gorm:"type:varchar(50);column:version;not null" json:"version"`    // Version of the workflow template
	Nodes   []GraphNode `gorm:"type:jsonb;column:nodes;serializer:json;not null" json:"nodes"` // List of nodes in the workflow graph
}

func (w *HSCodeWorkflow) TableName() string {
	return "hscode_workflows"
}
