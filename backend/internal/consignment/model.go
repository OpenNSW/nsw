package consignment

import (
	"fmt"
	"time"

	"github.com/OpenNSW/nsw/internal/hscode"
	"github.com/OpenNSW/nsw/internal/profile/cha"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// Flow represents the flow type of consignment.
// Keep values in sync with workflow/model.ConsignmentFlow — the workflow
// package keeps its own copy to avoid importing this package.
type Flow string

const (
	FlowImport Flow = "IMPORT"
	FlowExport Flow = "EXPORT"
)

// State represents the state of a consignment.
type State string

const (
	Initialized State = "INITIALIZED"
	InProgress  State = "IN_PROGRESS"
	Finished    State = "FINISHED"
)

// Consignment represents a consignment in the system.
type Consignment struct {
	ID        string    `gorm:"type:text;column:id;primaryKey;not null" json:"id"`
	CreatedAt time.Time `gorm:"type:timestamptz;column:created_at;not null;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:timestamptz;column:updated_at;not null;autoUpdateTime" json:"updatedAt"`

	Flow     Flow   `gorm:"type:varchar(50);column:flow;not null" json:"flow"`             // e.g., IMPORT, EXPORT
	TraderID string `gorm:"type:varchar(100);column:trader_id;not null" json:"traderId"`   // ID of the trader associated with the consignment
	State    State  `gorm:"type:varchar(50);column:state;not null" json:"state"`           // State of the consignment
	Items    []Item `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"` // Items in the consignment

	// CHA (Customs House Agent) – set at Stage 1 by Trader; CHA completes Stage 2 by selecting HS Code
	CHAID string     `gorm:"type:text;column:cha_id;not null" json:"chaId"` // Assigned CHA (Stage 1)
	CHA   cha.Record `gorm:"foreignKey:CHAID" json:"cha"`                   // Associated CHA entity

	// Relationships
	Workflow *model.Workflow `gorm:"foreignKey:ID;references:ID" json:"-"` // Associated Workflow (1:1, same ID)
}

func (c *Consignment) TableName() string {
	return "consignments"
}

// Item represents an individual item within a consignment.
type Item struct {
	HSCodeID string `gorm:"type:text;column:hs_code_id;not null" json:"hsCodeId"` // HS Code ID
}

// ItemResponseDTO represents an individual item in the consignment response.
type ItemResponseDTO struct {
	HSCode hscode.ResponseDTO `json:"hsCode"` // Full HS Code details
}

// InitializeConsignmentDTO is the request body for PUT /consignments/{id}/initialize (Stage 2 – CHA selects HS Code(s)).
type InitializeConsignmentDTO struct {
	HSCodeIDs []string `json:"hsCodeIds" binding:"required,min=1"`
}

// CreateConsignmentItemDTO represents the data required to create a consignment item.
type CreateConsignmentItemDTO struct {
	HSCodeID string `json:"hsCodeId" binding:"required"` // HS Code ID
}

// CreateConsignmentDTO represents the data required to create a consignment.
// Stage 1 (two-stage flow): provide flow + chaId only → creates shell with state INITIALIZED.
// Legacy / single-stage: provide flow + items → creates consignment and initializes workflow.
type CreateConsignmentDTO struct {
	Flow  Flow                       `json:"flow"`
	ChaID string                     `json:"chaId"`
	Items []CreateConsignmentItemDTO `json:"items,omitempty"`
}

func (d *CreateConsignmentDTO) Validate() error {
	if d.ChaID == "" {
		return fmt.Errorf("chaId is required")
	}
	if d.Flow != FlowImport && d.Flow != FlowExport {
		return fmt.Errorf("flow must be IMPORT or EXPORT")
	}
	return nil
}

// UpdateDTO represents the data required to update a consignment.
type UpdateDTO struct {
	ConsignmentID         string         `json:"consignmentId" binding:"required"` // Consignment ID
	State                 *State         `json:"state,omitempty"`                  // New state of the consignment (optional)
	AppendToGlobalContext map[string]any `json:"appendToGlobalContext,omitempty"`  // Additional global context to append to the consignment (optional)
}

// DetailDTO represents the full consignment data returned in detailed responses.
type DetailDTO struct {
	ID            string                          `json:"id"`            // Consignment ID
	Flow          Flow                            `json:"flow"`          // e.g., IMPORT, EXPORT
	TraderID      string                          `json:"traderId"`      // ID of the trader associated with the consignment
	ChaID         string                          `json:"chaId"`         // Assigned CHA (Stage 1)
	State         State                           `json:"state"`         // State of the consignment
	Items         []ItemResponseDTO               `json:"items"`         // Items in the consignment with full HS Code details
	CreatedAt     string                          `json:"createdAt"`     // Timestamp of consignment creation
	UpdatedAt     string                          `json:"updatedAt"`     // Timestamp of last consignment update
	WorkflowNodes []model.WorkflowNodeResponseDTO `json:"workflowNodes"` // Associated workflow nodes with template details
	Edges         []model.WorkflowEdgeResponseDTO `json:"edges"`         // Edges between workflow nodes
}

// SummaryDTO represents the consignment data returned in list responses.
type SummaryDTO struct {
	ID                         string            `json:"id"`                         // Consignment ID
	Flow                       Flow              `json:"flow"`                       // e.g., IMPORT, EXPORT
	TraderID                   string            `json:"traderId"`                   // ID of the trader associated with the consignment
	ChaID                      string            `json:"chaId"`                      // Assigned CHA (Stage 1)
	State                      State             `json:"state"`                      // State of the consignment
	Items                      []ItemResponseDTO `json:"items"`                      // Items in the consignment with full HS Code details
	CreatedAt                  string            `json:"createdAt"`                  // Timestamp of consignment creation
	UpdatedAt                  string            `json:"updatedAt"`                  // Timestamp of last consignment update
	WorkflowNodeCount          int               `json:"workflowNodeCount"`          // Total number of workflow nodes
	CompletedWorkflowNodeCount int               `json:"completedWorkflowNodeCount"` // Number of completed workflow nodes
}

// ListResult represents the result of querying consignments with pagination
type ListResult struct {
	TotalCount int64        `json:"totalCount"`
	Items      []SummaryDTO `json:"items"`
	Offset     int          `json:"offset"`
	Limit      int          `json:"limit"`
}

// Filter will be used when querying consignments as batch.
// For GET /consignments?role=trader use TraderID; for role=cha use ChaID (e.g. from query param cha_id).
type Filter struct {
	TraderID *string `json:"traderId,omitempty"`
	ChaID    *string `json:"chaId,omitempty"`
	Flow     *Flow   `json:"flow,omitempty"`
	State    *State  `json:"state,omitempty"`
	Offset   *int    `json:"offset,omitempty"`
	Limit    *int    `json:"limit,omitempty"`
}
