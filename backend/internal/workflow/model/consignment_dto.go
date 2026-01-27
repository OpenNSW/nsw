package model

import (
	"time"

	"github.com/google/uuid"
)

// CreateWorkflowForItemDTO represents the data required to create a workflow for an individual item within a consignment.
type CreateWorkflowForItemDTO struct {
	HSCodeID uuid.UUID   `json:"hsCodeId" binding:"required"` // HS Code ID of the item
	ItemData interface{} `json:"itemData,omitempty"`          // Additional item data (optional)
}

// CreateConsignmentDTO is the data transfer object for creating a new consignment.
type CreateConsignmentDTO struct {
	TradeFlow     TradeFlow                  `json:"tradeFlow" binding:"required,oneof=IMPORT EXPORT"` // Type of trade flow: IMPORT, EXPORT
	Items         []CreateWorkflowForItemDTO `json:"items" binding:"required,dive,required"`           // List of items in the consignment
	TraderID      *string                    `json:"traderId,omitempty"`                               // Reference to the Trader (optional: If not provided, use from auth context)
	GlobalContext map[string]interface{}     `json:"globalContext,omitempty"`                          // Global context for the consignment (optional)
}

// ConsignmentResponse represents the response data for a consignment.
type ConsignmentResponse struct {
	ID        uuid.UUID        `json:"id"`        // Consignment ID
	TradeFlow TradeFlow        `json:"tradeFlow"` // Type of trade flow: IMPORT or EXPORT
	Items     []Item           `json:"items"`     // List of items in the consignment
	TraderID  string           `json:"traderId"`  // Reference to the Trader
	State     ConsignmentState `json:"state"`     // IN_PROGRESS, REQUIRES_REWORK, FINISHED
	CreatedAt string           `json:"createdAt"` // Timestamp of consignment creation
	UpdatedAt string           `json:"updatedAt"` // Timestamp of last consignment update
}

// ToConsignmentResponse converts a Consignment model to a ConsignmentResponse DTO.
func (c *Consignment) ToConsignmentResponse() ConsignmentResponse {
	return ConsignmentResponse{
		ID:        c.ID,
		TradeFlow: c.TradeFlow,
		Items:     c.Items,
		TraderID:  c.TraderID,
		State:     c.State,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

// ConsignmentListResponseDTO represents the response data for a consignment.
type ConsignmentListResponseDTO struct {
	TotalCount int64                 `json:"totalCount"`
	Items      []ConsignmentResponse `json:"items"`
	Offset     int                   `json:"offset"`
	Limit      int                   `json:"limit"`
}

// ConsignmentTraderFilter will be used when querying as batch
type ConsignmentTraderFilter struct {
	TraderID string `json:"traderId,omitempty"`
	Offset   *int   `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty"`
}

// StepStatusUpdateDTO represents the data required to update the status of a step within a consignment.
type StepStatusUpdateDTO struct {
	StepID    string     `json:"stepId" binding:"required"`                                               // Step ID to be updated
	NewStatus TaskStatus `json:"newStatus" binding:"required,oneof=READY IN_PROGRESS COMPLETED REJECTED"` // New status for the step
}
