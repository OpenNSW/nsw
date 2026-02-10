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
	Name        string          `gorm:"type:varchar(255);column:name;not null" json:"name"`                      // Human-readable name of the workflow node template
	Description string          `gorm:"type:text;column:description" json:"description"`                         // Optional description of the workflow node template
	Type        taskPlugin.Type `gorm:"type:varchar(50);column:type;not null" json:"type"`                       // Type of the workflow node
	Config      json.RawMessage `gorm:"type:jsonb;column:config;not null;serializer:json" json:"config"`         // Configuration specific to the workflow node type
	DependsOn   UUIDArray       `gorm:"type:jsonb;column:depends_on;not null;serializer:json" json:"depends_on"` // Array of workflow node template IDs this node depends on
}

func (wnt *WorkflowNodeTemplate) TableName() string {
	return "workflow_node_templates"
}

// WorkflowNode represents an instance of a workflow node within a workflow.
type WorkflowNode struct {
	BaseModel
	ConsignmentID          *uuid.UUID        `gorm:"type:uuid;column:consignment_id" json:"consignmentId"`                              // Reference to the Consignment, Null if PreConsignment nodes
	PreConsignmentID       *uuid.UUID        `gorm:"type:uuid;column:pre_consignment_id" json:"preConsignmentId"`                       // Reference to the PreConsignment, Null if Consignment nodes
	WorkflowNodeTemplateID uuid.UUID         `gorm:"type:uuid;column:workflow_node_template_id;not null" json:"workflowNodeTemplateId"` // Reference to the WorkflowNodeTemplate
	State                  WorkflowNodeState `gorm:"type:varchar(50);column:state;not null" json:"state"`                               // State of the workflow node
	ExtendedState          *string           `gorm:"type:text;column:extended_state" json:"extendedState"`                              // Optional extended state information (e.g., error details)
	DependsOn              UUIDArray         `gorm:"type:jsonb;column:depends_on;not null;serializer:json" json:"depends_on"`           // Array of workflow node IDs this node depends on

	// Relationships
	Consignment          *Consignment         `gorm:"foreignKey:ConsignmentID;references:ID" json:"-"`                             // Associated Consignment
	PreConsignment       *PreConsignment      `gorm:"foreignKey:PreConsignmentID;references:ID" json:"-"`                          // Associated PreConsignment
	WorkflowNodeTemplate WorkflowNodeTemplate `gorm:"foreignKey:WorkflowNodeTemplateID;references:ID" json:"workflowNodeTemplate"` // Associated WorkflowNodeTemplate
}

func (wn *WorkflowNode) TableName() string {
	return "workflow_nodes"
}

// UpdateWorkflowNodeDTO is used to update the state of a workflow node.
type UpdateWorkflowNodeDTO struct {
	WorkflowNodeID      uuid.UUID         `json:"workflowNodeId" binding:"required"` // Workflow Node ID
	State               WorkflowNodeState `json:"state"`                             // New state of the workflow node
	AppendGlobalContext map[string]any    `json:"appendGlobalContext,omitempty"`     // Additional global context to append to the consignment (optional)
	ExtendedState       *string           `json:"extendedState,omitempty"`           // Optional extended state information (e.g., error details)
}

// WorkflowNodeResponseDTO represents a workflow node in the response.
type WorkflowNodeResponseDTO struct {
	ID                   uuid.UUID                       `json:"id"`                      // Workflow Node ID
	CreatedAt            string                          `json:"createdAt"`               // Timestamp of node creation
	UpdatedAt            string                          `json:"updatedAt"`               // Timestamp of last node update
	WorkflowNodeTemplate WorkflowNodeTemplateResponseDTO `json:"workflowNodeTemplate"`    // Workflow node template details
	State                WorkflowNodeState               `json:"state"`                   // State of the workflow node
	ExtendedState        *string                         `json:"extendedState,omitempty"` // Optional extended state information (e.g., error details)
	DependsOn            []uuid.UUID                     `json:"depends_on"`              // Array of workflow node IDs this node depends on
}

// WorkflowNodeTemplateResponseDTO represents workflow node template details in the response.
type WorkflowNodeTemplateResponseDTO struct {
	Name        string `json:"name"`        // Name of the workflow node template
	Description string `json:"description"` // Description of the workflow node template
	Type        string `json:"type"`        // Type of the workflow node template
}
