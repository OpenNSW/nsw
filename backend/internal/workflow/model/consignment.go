package model

import "github.com/google/uuid"

// ConsignmentFlow represents the flow type of a consignment.
type ConsignmentFlow string

const (
	ConsignmentFlowImport ConsignmentFlow = "IMPORT"
	ConsignmentFlowExport ConsignmentFlow = "EXPORT"
)

// ConsignmentState represents the state of a consignment.
type ConsignmentState string

const (
	ConsignmentStateInProgress ConsignmentState = "IN_PROGRESS"
	ConsignmentStateFinished   ConsignmentState = "FINISHED"
)

// Consignment represents a consignment in the system.
type Consignment struct {
	BaseModel
	Flow          ConsignmentFlow   `gorm:"type:varchar(50);column:flow;not null" json:"flow"`                              // e.g., IMPORT, EXPORT
	TraderID      string            `gorm:"type:varchar(100);column:trader_id;not null" json:"traderId"`                    // ID of the trader associated with the consignment
	State         ConsignmentState  `gorm:"type:varchar(50);column:state;not null" json:"state"`                            // State of the consignment
	Items         []ConsignmentItem `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"`                  // Items in the consignment
	GlobalContext map[string]any    `gorm:"type:jsonb;column:global_context;serializer:json;not null" json:"globalContext"` // Global context for the consignment

	// Relationships
	WorkflowNodes []WorkflowNode `gorm:"foreignKey:ConsignmentID;references:ID" json:"-"` // Associated WorkflowNodes
}

func (c *Consignment) TableName() string {
	return "consignments"
}

// ConsignmentItem represents an individual item within a consignment.
type ConsignmentItem struct {
	HSCodeID uuid.UUID `gorm:"type:uuid;column:hs_code_id;not null" json:"hsCodeId"` // HS Code ID
}

// ConsignmentItemResponseDTO represents an individual item in the consignment response.
type ConsignmentItemResponseDTO struct {
	HSCode HSCodeResponseDTO `json:"hsCode"` // Full HS Code details
}

// HSCodeResponseDTO represents HS Code details in the response.
type HSCodeResponseDTO struct {
	HSCodeID    uuid.UUID `json:"hsCodeId"`    // HS Code ID
	HSCode      string    `json:"hsCode"`      // HS Code
	Description string    `json:"description"` // Description of the HS Code
	Category    string    `json:"category"`    // Category of the HS Code
}

// CreateConsignmentItemDTO represents the data required to create a consignment item.
type CreateConsignmentItemDTO struct {
	HSCodeID uuid.UUID `json:"hsCodeId" binding:"required"` // HS Code ID
}

// CreateConsignmentDTO represents the data required to create a consignment.
type CreateConsignmentDTO struct {
	Flow  ConsignmentFlow            `json:"flow" binding:"required,oneof=IMPORT EXPORT"` // e.g., IMPORT, EXPORT
	Items []CreateConsignmentItemDTO `json:"items" binding:"required,dive,required"`      // Items in the consignment
}

// UpdateConsignmentDTO represents the data required to update a consignment.
type UpdateConsignmentDTO struct {
	ConsignmentID         uuid.UUID         `json:"consignmentId" binding:"required"` // Consignment ID
	State                 *ConsignmentState `json:"state,omitempty"`                  // New state of the consignment (optional)
	AppendToGlobalContext map[string]any    `json:"appendToGlobalContext,omitempty"`  // Additional global context to append to the consignment (optional)
}

// ConsignmentDetailDTO represents the full consignment data returned in detailed responses.
type ConsignmentDetailDTO struct {
	ID            uuid.UUID                    `json:"id"`            // Consignment ID
	Flow          ConsignmentFlow              `json:"flow"`          // e.g., IMPORT, EXPORT
	TraderID      string                       `json:"traderId"`      // ID of the trader associated with the consignment
	State         ConsignmentState             `json:"state"`         // State of the consignment
	Items         []ConsignmentItemResponseDTO `json:"items"`         // Items in the consignment with full HS Code details
	CreatedAt     string                       `json:"createdAt"`     // Timestamp of consignment creation
	UpdatedAt     string                       `json:"updatedAt"`     // Timestamp of last consignment update
	WorkflowNodes []WorkflowNodeResponseDTO    `json:"workflowNodes"` // Associated workflow nodes with template details
}

// ConsignmentSummaryDTO represents the consignment data returned in list responses.
type ConsignmentSummaryDTO struct {
	ID                         uuid.UUID                    `json:"id"`                         // Consignment ID
	Flow                       ConsignmentFlow              `json:"flow"`                       // e.g., IMPORT, EXPORT
	TraderID                   string                       `json:"traderId"`                   // ID of the trader associated with the consignment
	State                      ConsignmentState             `json:"state"`                      // State of the consignment
	Items                      []ConsignmentItemResponseDTO `json:"items"`                      // Items in the consignment with full HS Code details
	CreatedAt                  string                       `json:"createdAt"`                  // Timestamp of consignment creation
	UpdatedAt                  string                       `json:"updatedAt"`                  // Timestamp of last consignment update
	WorkflowNodeCount          int                          `json:"workflowNodeCount"`          // Total number of workflow nodes
	CompletedWorkflowNodeCount int                          `json:"completedWorkflowNodeCount"` // Number of completed workflow nodes
}

// ConsignmentListResult represents the result of querying consignments with pagination
type ConsignmentListResult struct {
	TotalCount int64                   `json:"totalCount"`
	Items      []ConsignmentSummaryDTO `json:"items"`
	Offset     int                     `json:"offset"`
	Limit      int                     `json:"limit"`
}

// ConsignmentFilter will be used when querying consignments as batch
type ConsignmentFilter struct {
	TraderID *string           `json:"traderId,omitempty"`
	Flow     *ConsignmentFlow  `json:"flow,omitempty"`
	State    *ConsignmentState `json:"state,omitempty"`
	Offset   *int              `json:"offset,omitempty"`
	Limit    *int              `json:"limit,omitempty"`
}
