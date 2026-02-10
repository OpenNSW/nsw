package model

import "github.com/google/uuid"

type PreConsignmentState string

const (
	PreConsignmentStateLocked     PreConsignmentState = "LOCKED"      // Pre-consignment is locked and cannot be processed because previous steps are incomplete
	PreConsignmentStateReady      PreConsignmentState = "READY"       // Pre-consignment is ready to be processed
	PreConsignmentStateInProgress PreConsignmentState = "IN_PROGRESS" // Pre-consignment is currently being processed
	PreConsignmentStateCompleted  PreConsignmentState = "COMPLETED"   // Pre-consignment has been completed
)

type PreConsignmentTemplate struct {
	BaseModel
	Name               string    `gorm:"type:varchar(255);column:name;not null" json:"name"`            // Human-readable name of the pre-consignment template
	Description        string    `gorm:"type:text;column:description" json:"description"`               // Optional description of the pre-consignment template
	WorkflowTemplateID uuid.UUID `json:"workflowTemplateId"`                                            // ID of the workflow template to use for this pre-consignment
	DependsOn          []string  `gorm:"type:jsonb;column:depends_on;serializer:json" json:"dependsOn"` // List of pre-consignment template IDs that this pre-consignment template depends on
}

func (pct *PreConsignmentTemplate) TableName() string {
	return "pre_consignment_templates"
}

type PreConsignment struct {
	BaseModel
	TraderID                 string              `gorm:"type:varchar(255);not null" json:"traderId"`
	PreConsignmentTemplateID uuid.UUID           `gorm:"type:uuid;not null" json:"preConsignmentTemplateId"`
	State                    PreConsignmentState `gorm:"type:varchar(50);not null" json:"state"`
	TraderContext            map[string]any      `gorm:"type:jsonb;column:trader_context;serializer:json;not null" json:"traderContext"` // Context specific to the trader

	// Relationships
	PreConsignmentTemplate PreConsignmentTemplate `gorm:"foreignKey:PreConsignmentTemplateID;references:ID" json:"-"` // Associated PreConsignmentTemplate
	WorkflowNodes          []WorkflowNode         `gorm:"foreignKey:PreConsignmentID;references:ID" json:"-"`         // Associated WorkflowNodes
}

func (pc *PreConsignment) TableName() string {
	return "pre_consignments"
}

// CreatePreConsignmentDTO is used to create a new pre-consignment.
type CreatePreConsignmentDTO struct {
	TraderID                 string    `json:"traderId" validate:"required"`                 // Trader ID associated with the pre-consignment
	PreConsignmentTemplateID uuid.UUID `json:"preConsignmentTemplateId" validate:"required"` // ID of the pre-consignment template to use
}

// UpdatePreConsignmentStateDTO is used to update the state of a pre-consignment.
type UpdatePreConsignmentStateDTO struct {
	State         PreConsignmentState `json:"state" validate:"required"` // New state of the pre-consignment
	TraderContext map[string]any      `json:"traderContext"`             // Optional updated trader context
}

// PreConsignmentTemplateResponseDTO represents a pre-consignment template in the response.
type PreConsignmentTemplateResponseDTO struct {
	ID          uuid.UUID `json:"id"`          // Template ID
	Name        string    `json:"name"`        // Human-readable name
	Description string    `json:"description"` // Description of the template
	DependsOn   []string  `json:"dependsOn"`   // List of dependency template IDs
}

// TraderPreConsignmentResponseDTO represents a pre-consignment template in the response.
type TraderPreConsignmentResponseDTO struct {
	ID               uuid.UUID           `json:"id"`                         // Template ID
	PreConsignmentID *uuid.UUID          `json:"preConsignmentId,omitempty"` // Pre-consignment ID (if applicable)
	Name             string              `json:"name"`                       // Human-readable name
	Description      string              `json:"description"`                // Description of the template
	DependsOn        []string            `json:"dependsOn"`                  // List of dependency template IDs
	State            PreConsignmentState `json:"state"`                      // Computed state: if PreConsignment Instance is there, use its PreConsignmentState; if not, READY if dependencies are met, LOCKED otherwise

	// Relationships
	PreConsignment *PreConsignment `json:"preConsignment,omitempty"` // Associated PreConsignment instance (if it exists)
}

// TraderPreConsignmentsResponseDTO represents a list of pre-consignment templates for a trader in the response.
type TraderPreConsignmentsResponseDTO struct {
	TotalCount int64                             `json:"totalCount"` // Total number of pre-consignment templates for the trader
	Items      []TraderPreConsignmentResponseDTO `json:"items"`      // List of pre-consignment templates for the trader
	Offset     int64                             `json:"offset"`     // Pagination offset
	Limit      int64                             `json:"limit"`      // Pagination limit
}

// PreConsignmentResponseDTO represents a pre-consignment in the response.
type PreConsignmentResponseDTO struct {
	ID                     uuid.UUID                         `json:"id"`                     // Pre-consignment ID
	TraderID               string                            `json:"traderId"`               // Trader ID associated with the pre-consignment
	State                  PreConsignmentState               `json:"state"`                  // State of the pre-consignment
	TraderContext          map[string]any                    `json:"traderContext"`          // Trader-specific context
	CreatedAt              string                            `json:"createdAt"`              // Timestamp of creation
	UpdatedAt              string                            `json:"updatedAt"`              // Timestamp of last update
	PreConsignmentTemplate PreConsignmentTemplateResponseDTO `json:"preConsignmentTemplate"` // Template details
	WorkflowNodes          []WorkflowNodeResponseDTO         `json:"workflowNodes"`          // Associated workflow nodes
}
