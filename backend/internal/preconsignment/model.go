package preconsignment

import "github.com/OpenNSW/nsw/internal/workflow/model"

type State string

const (
	StateLocked     State = "LOCKED"      // Pre-consignment is locked and cannot be processed because previous steps are incomplete
	StateReady      State = "READY"       // Pre-consignment is ready to be processed
	StateInProgress State = "IN_PROGRESS" // Pre-consignment is currently being processed
	StateCompleted  State = "COMPLETED"   // Pre-consignment has been completed
)

type Template struct {
	model.BaseModel
	Name               string   `gorm:"type:varchar(255);column:name;not null" json:"name"`            // Human-readable name of the pre-consignment template
	Description        string   `gorm:"type:text;column:description" json:"description"`               // Optional description of the pre-consignment template
	WorkflowTemplateID string   `json:"workflowTemplateId"`                                            // ID of the workflow template to use for this pre-consignment
	DependsOn          []string `gorm:"type:jsonb;column:depends_on;serializer:json" json:"dependsOn"` // List of pre-consignment template IDs that this pre-consignment template depends on
}

func (pct *Template) TableName() string {
	return "pre_consignment_templates"
}

type PreConsignment struct {
	model.BaseModel
	TraderID                 string `gorm:"type:varchar(255);not null" json:"traderId"`
	PreConsignmentTemplateID string `gorm:"type:text;not null" json:"preConsignmentTemplateId"`
	State                    State  `gorm:"type:varchar(50);not null" json:"state"`

	// Relationships
	PreConsignmentTemplate Template        `gorm:"foreignKey:PreConsignmentTemplateID;references:ID" json:"-"` // Associated PreConsignmentTemplate
	Workflow               *model.Workflow `gorm:"foreignKey:ID;references:ID" json:"-"`                       // Associated Workflow (1:1, same ID)
}

func (pc *PreConsignment) TableName() string {
	return "pre_consignments"
}

// CreatePreConsignmentDTO is used to create a new pre-consignment.
type CreatePreConsignmentDTO struct {
	PreConsignmentTemplateID string `json:"preConsignmentTemplateId" validate:"required"` // ID of the pre-consignment template to use
}

// UpdatePreConsignmentStateDTO is used to update the state of a pre-consignment.
type UpdatePreConsignmentStateDTO struct {
	State         State          `json:"state" validate:"required"` // New state of the pre-consignment
	TraderContext map[string]any `json:"traderContext"`             // Optional updated trader context
}

// TemplateResponseDTO represents a pre-consignment template in the response.
type TemplateResponseDTO struct {
	ID          string   `json:"id"`          // Template ID
	Name        string   `json:"name"`        // Human-readable name
	Description string   `json:"description"` // Description of the template
	DependsOn   []string `json:"dependsOn"`   // List of dependency template IDs
}

// TraderPreConsignmentResponseDTO represents a pre-consignment template in the response.
type TraderPreConsignmentResponseDTO struct {
	ID               string   `json:"id"`                         // Template ID
	PreConsignmentID *string  `json:"preConsignmentId,omitempty"` // Pre-consignment ID (if applicable)
	Name             string   `json:"name"`                       // Human-readable name
	Description      string   `json:"description"`                // Description of the template
	DependsOn        []string `json:"dependsOn"`                  // List of dependency template IDs
	State            State    `json:"state"`                      // Computed state: if PreConsignment Instance is there, use its PreConsignmentState; if not, READY if dependencies are met, LOCKED otherwise

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

// ResponseDTO represents a pre-consignment in the response.
type ResponseDTO struct {
	ID                     string                          `json:"id"`                     // Pre-consignment ID
	TraderID               string                          `json:"traderId"`               // Trader ID associated with the pre-consignment
	State                  State                           `json:"state"`                  // State of the pre-consignment
	TraderContext          map[string]any                  `json:"traderContext"`          // Trader-specific context
	CreatedAt              string                          `json:"createdAt"`              // Timestamp of creation
	UpdatedAt              string                          `json:"updatedAt"`              // Timestamp of last update
	PreConsignmentTemplate TemplateResponseDTO             `json:"preConsignmentTemplate"` // Template details
	WorkflowNodes          []model.WorkflowNodeResponseDTO `json:"workflowNodes"`          // Associated workflow nodes
}
