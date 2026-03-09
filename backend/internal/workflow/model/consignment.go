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
	ConsignmentStateAwaitingInitiation ConsignmentState = "AWAITING_INITIATION"
	ConsignmentStateInProgress         ConsignmentState = "IN_PROGRESS"
	ConsignmentStateFinished           ConsignmentState = "FINISHED"
)

// Consignment represents a consignment in the system.
type Consignment struct {
	BaseModel
	Flow          ConsignmentFlow   `gorm:"type:varchar(50);column:flow;not null" json:"flow"`                              // e.g., IMPORT, EXPORT
	TraderID      string            `gorm:"type:varchar(100);column:trader_id;not null" json:"traderId"`                    // ID of the trader associated with the consignment
	State         ConsignmentState  `gorm:"type:varchar(50);column:state;not null" json:"state"`                            // State of the consignment
	Items         []ConsignmentItem `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"`                  // Items in the consignment
	GlobalContext map[string]any    `gorm:"type:jsonb;column:global_context;serializer:json;not null" json:"globalContext"` // Global context for the consignment
	EndNodeID     *uuid.UUID        `gorm:"type:uuid;column:end_node_id" json:"endNodeId,omitempty"`                        // Optional reference to the end workflow node, used for quick lookup of completion status

	// CHA (Customs House Agent) – set at Stage 1 by Trader; CHA completes Stage 2 by selecting HS Code
	CHAID *uuid.UUID `gorm:"type:uuid;column:cha_id" json:"chaId,omitempty"`
	CHA   *CHA       `gorm:"foreignKey:CHAID" json:"cha,omitempty"`

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

// InitializeConsignmentDTO is the request body for PUT /consignments/{id}/initialize (Stage 2 – CHA selects HS Code).
type InitializeConsignmentDTO struct {
	HSCodeID uuid.UUID `json:"hsCodeId" binding:"required"` // HS Code ID to start the workflow
}

// CreateConsignmentItemDTO represents the data required to create a consignment item.
type CreateConsignmentItemDTO struct {
	HSCodeID uuid.UUID `json:"hsCodeId" binding:"required"` // HS Code ID
}

// CreateConsignmentDTO represents the data required to create a consignment.
// Stage 1 (two-stage flow): provide flow + chaId only → creates shell with state AWAITING_INITIATION.
// Legacy / single-stage: provide flow + items → creates consignment and initializes workflow.
type CreateConsignmentDTO struct {
	Flow  ConsignmentFlow            `json:"flow" binding:"required,oneof=IMPORT EXPORT"` // e.g., IMPORT, EXPORT
	ChaID *uuid.UUID                 `json:"chaId,omitempty"`                             // Stage 1: assign CHA (shell only)
	Items []CreateConsignmentItemDTO `json:"items,omitempty"`                             // Legacy: HS code items; when ChaID is set, items are ignored
}

// UpdateConsignmentDTO represents the data required to update a consignment.
type UpdateConsignmentDTO struct {
	ConsignmentID         uuid.UUID         `json:"consignmentId" binding:"required"` // Consignment ID
	State                 *ConsignmentState `json:"state,omitempty"`                  // New state of the consignment (optional)
	AppendToGlobalContext map[string]any    `json:"appendToGlobalContext,omitempty"`  // Additional global context to append to the consignment (optional)
}

// ConsignmentDetailDTO represents the full consignment data returned in detailed responses.
type ConsignmentDetailDTO struct {
	ID            uuid.UUID                    `json:"id"`              // Consignment ID
	Flow          ConsignmentFlow              `json:"flow"`            // e.g., IMPORT, EXPORT
	TraderID      string                       `json:"traderId"`        // ID of the trader associated with the consignment
	ChaID         *uuid.UUID                   `json:"chaId,omitempty"` // Assigned CHA (Stage 1)
	State         ConsignmentState             `json:"state"`           // State of the consignment
	Items         []ConsignmentItemResponseDTO `json:"items"`           // Items in the consignment with full HS Code details
	CreatedAt     string                       `json:"createdAt"`       // Timestamp of consignment creation
	UpdatedAt     string                       `json:"updatedAt"`       // Timestamp of last consignment update
	WorkflowNodes []WorkflowNodeResponseDTO    `json:"workflowNodes"`   // Associated workflow nodes with template details
}

// ConsignmentSummaryDTO represents the consignment data returned in list responses.
type ConsignmentSummaryDTO struct {
	ID                         uuid.UUID                    `json:"id"`                         // Consignment ID
	Flow                       ConsignmentFlow              `json:"flow"`                       // e.g., IMPORT, EXPORT
	TraderID                   string                       `json:"traderId"`                   // ID of the trader associated with the consignment
	ChaID                      *uuid.UUID                   `json:"chaId,omitempty"`            // Assigned CHA (Stage 1)
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

// ConsignmentFilter will be used when querying consignments as batch.
// For GET /consignments?role=trader use TraderID; for role=cha use ChaID (e.g. from query param cha_id).
type ConsignmentFilter struct {
	TraderID *string           `json:"traderId,omitempty"`
	ChaID    *uuid.UUID        `json:"chaId,omitempty"`
	Flow     *ConsignmentFlow  `json:"flow,omitempty"`
	State    *ConsignmentState `json:"state,omitempty"`
	Offset   *int              `json:"offset,omitempty"`
	Limit    *int              `json:"limit,omitempty"`
}
