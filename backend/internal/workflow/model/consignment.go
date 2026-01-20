package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Consignment represents the state and data of a consignment in the workflow system.
type Consignment struct {
	BaseModel
	Type     ConsignmentType  `gorm:"type:varchar(20);column:type;not null" json:"type"`             // Type of consignment: IMPORT, EXPORT
	Items    []Item           `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"` // List of items in the consignment
	TraderID string           `gorm:"type:varchar(255);column:trader_id;not null" json:"traderId"`   // Reference to the Trader
	State    ConsignmentState `gorm:"type:varchar(20);column:state;not null" json:"state"`           // IN_PROGRESS, FINISHED
}

func (c *Consignment) TableName() string {
	return "consignments"
}

// Item represents an individual item within a consignment.
type Item struct {
	HSCode             string          `json:"hsCode"`             // HS Code of the item
	Metadata           json.RawMessage `json:"metadata"`           // Information about the item such as description, quantity, value, etc.
	WorkflowTemplateID uuid.UUID       `json:"workflowTemplateId"` // Workflow Template ID associated with this item
	Tasks              []uuid.UUID     `json:"tasks"`              // List of task IDs associated with this item
}
