package model

import (
	"encoding/json"

	"github.com/google/uuid"

	taskPlugin "github.com/OpenNSW/nsw/internal/task/plugin"
)

type WorkflowNodeType string

const (
	WorkflowNodeTypeSimpleForm   WorkflowNodeType = "SIMPLE_FORM"    // Node for simple form submission
	WorkflowNodeTypeWaitForEvent WorkflowNodeType = "WAIT_FOR_EVENT" // Node that waits for an external event to occur
)

type WorkflowNodeState string

const (
	WorkflowNodeStateLocked     WorkflowNodeState = "LOCKED"      // Node is locked and cannot be activated because previous nodes are incomplete
	WorkflowNodeStateReady      WorkflowNodeState = "READY"       // Node is ready to be activated
	WorkflowNodeStateInProgress WorkflowNodeState = "IN_PROGRESS" // Node is currently active and in progress
	WorkflowNodeStateCompleted  WorkflowNodeState = "COMPLETED"   // Node has been completed
	WorkflowNodeStateFailed     WorkflowNodeState = "FAILED"      // Node has failed
)

// WorkflowNodeTemplate represents a template for a workflow node.
type WorkflowNodeTemplate struct {
	BaseModel
	Type      taskPlugin.Type `gorm:"type:varchar(50);column:type;not null" json:"type"`                       // Type of the workflow node
	Config    json.RawMessage `gorm:"type:jsonb;column:config;not null;serializer:json" json:"config"`         // Configuration specific to the workflow node type
	DependsOn UUIDArray       `gorm:"type:jsonb;column:depends_on;not null;serializer:json" json:"depends_on"` // Array of workflow node template IDs this node depends on
}

func (wnt *WorkflowNodeTemplate) TableName() string {
	return "workflow_node_templates"
}

// WorkflowNode represents an instance of a workflow node within a workflow.
type WorkflowNode struct {
	BaseModel
	ConsignmentID          uuid.UUID         `gorm:"type:uuid;column:consignment_id;not null" json:"consignmentId"`                     // Reference to the Consignment
	WorkflowNodeTemplateID uuid.UUID         `gorm:"type:uuid;column:workflow_node_template_id;not null" json:"workflowNodeTemplateId"` // Reference to the WorkflowNodeTemplate
	State                  WorkflowNodeState `gorm:"type:varchar(50);column:state;not null" json:"state"`                               // State of the workflow node
	DependsOn              UUIDArray         `gorm:"type:jsonb;column:depends_on;not null;serializer:json" json:"depends_on"`           // Array of workflow node IDs this node depends on

	// Relationships
	Consignment          Consignment          `gorm:"foreignKey:ConsignmentID;references:ID" json:"-"`          // Associated Consignment
	WorkflowNodeTemplate WorkflowNodeTemplate `gorm:"foreignKey:WorkflowNodeTemplateID;references:ID" json:"-"` // Associated WorkflowNodeTemplate
}

func (wn *WorkflowNode) TableName() string {
	return "workflow_nodes"
}

// UpdateWorkflowNodeDTO is used to update the state of a workflow node.
type UpdateWorkflowNodeDTO struct {
	WorkflowNodeID      uuid.UUID         `json:"workflowNodeId" binding:"required"` // Workflow Node ID
	State               WorkflowNodeState `json:"state"`                             // New state of the workflow node
	AppendGlobalContext map[string]any    `json:"appendGlobalContext,omitempty"`     // Additional global context to append to the consignment (optional)
}
