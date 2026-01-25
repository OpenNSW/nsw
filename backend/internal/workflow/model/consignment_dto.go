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
