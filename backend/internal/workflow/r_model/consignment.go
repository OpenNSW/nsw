package r_model

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
}

func (c *Consignment) TableName() string {
	return "consignments"
}

// ConsignmentItem represents an individual item within a consignment.
type ConsignmentItem struct {
	HSCodeID     uuid.UUID `gorm:"type:uuid;column:hs_code_id;not null" json:"hsCodeId"`         // HS Code ID
	ItemMetadata any       `gorm:"type:jsonb;column:item_metadata;not null" json:"itemMetadata"` // Metadata about the item
}

// CreateConsignmentItemDTO represents the data required to create a consignment item.
type CreateConsignmentItemDTO struct {
	HSCodeID     uuid.UUID `json:"hsCodeId" binding:"required"` // HS Code ID
	ItemMetadata any       `json:"itemMetadata,omitempty"`      // Metadata about the item (optional)
}

// CreateConsignmentDTO represents the data required to create a consignment.
type CreateConsignmentDTO struct {
	Flow          ConsignmentFlow            `json:"flow" binding:"required,oneof=IMPORT EXPORT"` // e.g., IMPORT, EXPORT
	TraderID      *string                    `json:"traderId,omitempty"`                          // ID of the trader associated with the consignment (optional)
	Items         []CreateConsignmentItemDTO `json:"items" binding:"required,dive,required"`      // Items in the consignment
	GlobalContext map[string]any             `json:"globalContext,omitempty"`                     // Global context for the consignment (optional)
}

// UpdateConsignmentDTO represents the data required to update a consignment.
type UpdateConsignmentDTO struct {
	ConsignmentID         uuid.UUID         `json:"consignmentId" binding:"required"` // Consignment ID
	State                 *ConsignmentState `json:"state,omitempty"`                  // New state of the consignment (optional)
	AppendToGlobalContext map[string]any    `json:"appendToGlobalContext,omitempty"`  // Additional global context to append to the consignment (optional)
}

// ConsignmentResponseDTO represents the consignment data returned in responses.
type ConsignmentResponseDTO struct {
	ID            uuid.UUID         `json:"id"`            // Consignment ID
	Flow          ConsignmentFlow   `json:"flow"`          // e.g., IMPORT, EXPORT
	TraderID      string            `json:"traderId"`      // ID of the trader associated with the consignment
	State         ConsignmentState  `json:"state"`         // State of the consignment
	Items         []ConsignmentItem `json:"items"`         // Items in the consignment
	GlobalContext map[string]any    `json:"globalContext"` // Global context for the consignment
	CreatedAt     string            `json:"createdAt"`     // Timestamp of consignment creation
	UpdatedAt     string            `json:"updatedAt"`     // Timestamp of last consignment update
	WorkflowNodes []WorkflowNode    `json:"workflowNodes"` // Associated workflow nodes
}

// ConsignmentListResult represents the result of querying consignments with pagination
type ConsignmentListResult struct {
	TotalCount int64                    `json:"totalCount"`
	Items      []ConsignmentResponseDTO `json:"items"`
	Offset     int                      `json:"offset"`
	Limit      int                      `json:"limit"`
}

// ConsignmentFilter will be used when querying consignments as batch
type ConsignmentFilter struct {
	TraderID *string           `json:"traderId,omitempty"`
	Flow     *ConsignmentFlow  `json:"flow,omitempty"`
	State    *ConsignmentState `json:"state,omitempty"`
	Offset   *int              `json:"offset,omitempty"`
	Limit    *int              `json:"limit,omitempty"`
}
