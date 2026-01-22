package model

import (
	"github.com/google/uuid"
)

// ConsignmentType represents the type of consignment.
type ConsignmentType string

const (
	ConsignmentTypeImport ConsignmentType = "IMPORT"
	ConsignmentTypeExport ConsignmentType = "EXPORT"
)

// ConsignmentState represents the state of a consignment in the workflow.
type ConsignmentState string

const (
	ConsignmentStateInProgress     ConsignmentState = "IN_PROGRESS"
	ConsignmentStateRequiresRework ConsignmentState = "REQUIRES_REWORK" // At least one task has been rejected
	ConsignmentStateFinished       ConsignmentState = "FINISHED"
)

// Consignment represents the state and data of a consignment in the workflow system.
type Consignment struct {
	BaseModel
	Type     ConsignmentType  `gorm:"type:varchar(20);column:type;not null" json:"type"`             // Type of consignment: IMPORT, EXPORT
	Items    []Item           `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"` // List of items in the consignment
	TraderID string           `gorm:"type:varchar(255);column:trader_id;not null" json:"traderId"`   // Reference to the Trader
	State    ConsignmentState `gorm:"type:varchar(20);column:state;not null" json:"state"`           // IN_PROGRESS, REQUIRES_REWORK, FINISHED
}

func (c *Consignment) TableName() string {
	return "consignments"
}

// Item represents an individual item within a consignment.
type Item struct {
	HSCode             string      `json:"hsCode"`             // HS Code of the item
	WorkflowTemplateID uuid.UUID   `json:"workflowTemplateId"` // Workflow Template ID associated with this item
	Tasks              []uuid.UUID `json:"tasks"`              // List of task IDs associated with this item
}
