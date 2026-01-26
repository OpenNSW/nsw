package model

import (
	"github.com/google/uuid"
)

// TradeFlow represents the trade flow of a consignment.
type TradeFlow string

const (
	TradeFlowImport TradeFlow = "IMPORT"
	TradeFlowExport TradeFlow = "EXPORT"
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
	TradeFlow TradeFlow        `gorm:"type:varchar(20);column:trade_flow;not null" json:"tradeFlow"`  // Type of trade flow: IMPORT or EXPORT
	Items     []Item           `gorm:"type:jsonb;column:items;serializer:json;not null" json:"items"` // List of items in the consignment
	TraderID  string           `gorm:"type:varchar(255);column:trader_id;not null" json:"traderId"`   // Reference to the Trader
	State     ConsignmentState `gorm:"type:varchar(20);column:state;not null" json:"state"`           // IN_PROGRESS, REQUIRES_REWORK, FINISHED
}

func (c *Consignment) TableName() string {
	return "consignments"
}

type ConsignmentStep struct {
	StepID    string     `json:"stepId"`    // Step ID within the workflow template
	Type      StepType   `json:"type"`      // Type of the task
	TaskID    uuid.UUID  `json:"taskId"`    // Associated Task ID
	Status    TaskStatus `json:"status"`    // Current status of the task
	DependsOn []string   `json:"dependsOn"` // List of step IDs that this step depends on

}

// Item represents an individual item within a consignment.
type Item struct {
	HSCodeID uuid.UUID         `json:"hsCodeID"` // HS Code ID of the item
	Steps    []ConsignmentStep `json:"steps"`    // List of steps associated with this item
}
