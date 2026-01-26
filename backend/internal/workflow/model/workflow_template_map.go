package model

import "github.com/google/uuid"

// WorkflowTemplateMap represents the mapping between HSCode and Workflow.
type WorkflowTemplateMap struct {
	BaseModel
	HSCodeID           uuid.UUID `gorm:"type:uuid;column:hs_code_id;not null" json:"hsCodeId"`
	TradeFlow          TradeFlow `gorm:"type:varchar(50);column:trade_flow;not null" json:"tradeFlow"` // e.g., IMPORT, EXPORT
	WorkflowTemplateID uuid.UUID `gorm:"type:uuid;column:workflow_template_id;not null" json:"workflowTemplateId"`

	// Relationships
	HSCode           HSCode           `gorm:"foreignKey:HSCodeID;references:ID" json:"hsCode"`
	WorkflowTemplate WorkflowTemplate `gorm:"foreignKey:WorkflowTemplateID;references:ID" json:"workflowTemplate"`
}

func (w *WorkflowTemplateMap) TableName() string {
	return "workflow_template_maps"
}
