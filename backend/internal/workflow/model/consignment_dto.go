package model

// CreateWorkflowForItemDTO represents the data required to create a workflow.json for an individual item within a consignment.
type CreateWorkflowForItemDTO struct {
	HSCode             string `json:"hsCode" binding:"required"`             // HS Code of the item
	WorkflowTemplateID string `json:"workflowTemplateId" binding:"required"` // Workflow Template ID associated with this item
}

// CreateConsignmentDTO is the data transfer object for creating a new consignment.
type CreateConsignmentDTO struct {
	Type     ConsignmentType            `json:"type" binding:"required,oneof=IMPORT EXPORT"` // Type of consignment: IMPORT, EXPORT
	Items    []CreateWorkflowForItemDTO `json:"items" binding:"required,dive,required"`      // List of items in the consignment
	TraderID string                     `json:"traderId" binding:"required"`                 // Reference to the Trader
}
