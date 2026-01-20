package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Consignment represents the state and data of a consignment in the workflow system.
type Consignment struct {
	BaseModel
	Type         ConsignmentType  `gorm:"type:varchar(20);column:type;not null" json:"type"`                  // Type of consignment: IMPORT, EXPORT
	Items        []Item           `gorm:"type:jsonb;column:items;not null" json:"items"`                      // List of items in the consignment
	TraderID     string           `gorm:"type:uuid;column:trader_id;not null" json:"traderId"`                // Reference to the Trader
	CurrentState ConsignmentState `gorm:"type:varchar(20);column:current_state;not null" json:"currentState"` // IN_PROGRESS, FINISHED
}

type Item struct {
	HSCode             string          `gorm:"type:varchar(50);column:hs_code;not null" json:"hsCode"`                   // HS Code of the item
	Metadata           json.RawMessage `gorm:"type:jsonb;column:metadata;not null" json:"metadata"`                      // Information about the item such as description, quantity, value, etc.
	WorkflowTemplateID uuid.UUID       `gorm:"type:uuid;column:workflow_template_id;not null" json:"workflowTemplateId"` // Workflow Template ID associated with this item
	Tasks              []uuid.UUID     `gorm:"type:uuid[];column:tasks;not null" json:"tasks"`                           // List of task IDs associated with this item
}
