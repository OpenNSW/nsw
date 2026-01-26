package model

import "github.com/google/uuid"

// CreateWorkflowForItemDTO represents the data required to create a workflow for an individual item within a consignment.
type CreateWorkflowForItemDTO struct {
	HSCodeID           uuid.UUID  `json:"hsCodeId" binding:"required"`  // HS Code ID of the item
	WorkflowTemplateID *uuid.UUID `json:"workflowTemplateId,omitempty"` // Workflow Template ID associated with this item (optional)
}

// CreateConsignmentDTO is the data transfer object for creating a new consignment.
type CreateConsignmentDTO struct {
	TradeFlow TradeFlow                  `json:"tradeFlow" binding:"required,oneof=IMPORT EXPORT"` // Type of trade flow: IMPORT, EXPORT
	Items     []CreateWorkflowForItemDTO `json:"items" binding:"required,dive,required"`           // List of items in the consignment
	TraderID  *string                    `json:"traderId,omitempty"`                               // Reference to the Trader
}

// ConsignmentResponse represents the response data for a consignment.
type ConsignmentResponse struct {
	ID        uuid.UUID        `json:"id"`        // Consignment ID
	TradeFlow TradeFlow        `json:"tradeFlow"` // Type of trade flow: IMPORT or EXPORT
	Items     []Item           `json:"items"`     // List of items in the consignment
	TraderID  string           `json:"traderId"`  // Reference to the Trader
	State     ConsignmentState `json:"state"`     // IN_PROGRESS, REQUIRES_REWORK, FINISHED
}

// ToConsignmentResponse converts a Consignment model to a ConsignmentResponse DTO.
func (c *Consignment) ToConsignmentResponse() ConsignmentResponse {
	return ConsignmentResponse{
		ID:        c.ID,
		TradeFlow: c.TradeFlow,
		Items:     c.Items,
		TraderID:  c.TraderID,
		State:     c.State,
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
