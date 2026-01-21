package task

import (
	"context"

	"github.com/google/uuid"
)

// WorkflowStep represents a step in the workflow process
type WorkflowStep struct {
	StepID     string                 `json:"stepId"`
	Status     string                 `json:"status"` // LOCKED, COMPLETED, READY
	Type       string                 `json:"type"`   // TRADER_FORM, OGA_APPLICATION, SYSTEM_WAIT, etc.
	Config     map[string]interface{} `json:"config"`
	DependsOn  []string               `json:"dependsOn"`  // List of stepIds that must complete first
	IsRealtime bool                   `json:"isRealtime"` // Indicates if task is realtime (default: true for TRADER_FORM, false for OGA_FORM)
}

// WorkflowMetadata represents the JSON structure of a workflow process
type WorkflowMetadata struct {
	WorkflowID string         `json:"workflowId"`
	Version    string         `json:"version"`
	Steps      []WorkflowStep `json:"steps"`
}

// WorkflowEngineClient provides access to workflow metadata (RESTful interface)
// Task Manager calls Workflow Engine API to get workflow information.
// This follows microservices architecture: Task Manager doesn't access Workflow Engine's database directly.
type WorkflowEngineClient interface {
	// GetWorkflowMetadata retrieves workflow metadata for a consignment
	GetWorkflowMetadata(ctx context.Context, consignmentID uuid.UUID) (*WorkflowMetadata, error)

	// GetWorkflowState retrieves current state of all steps for a consignment
	// Returns a map of stepId -> status (e.g., "cusdec" -> "COMPLETED")
	GetWorkflowState(ctx context.Context, consignmentID uuid.UUID) (map[string]string, error)

	// UpdateWorkflowState updates the workflow state for a task (async callback)
	// Called by Task Manager to notify Workflow Manager of task state changes
	// state should be one of: "INPROGRESS", "COMPLETED", "REJECTED"
	UpdateWorkflowState(ctx context.Context, taskID uuid.UUID, state string) error
}
