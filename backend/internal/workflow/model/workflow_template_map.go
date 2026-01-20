package model

import "github.com/google/uuid"

// HSCodeWorkflowMap represents the mapping between HSCode and Workflow.
type HSCodeWorkflowTemplateMap struct {
	HSCodeID           uuid.UUID `gorm:"type:uuid;column:hs_code_id;not null" json:"hsCodeId"`
	Type               string    `gorm:"type:varchar(50);column:type;not null" json:"type"` // e.g., IMPORT, EXPORT
	WorkflowTemplateID uuid.UUID `gorm:"type:uuid;column:workflow_id;not null" json:"workflowId"`
}
