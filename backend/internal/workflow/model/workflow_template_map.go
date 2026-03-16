package model

import "github.com/google/uuid"

// WorkflowTemplateMap represents the mapping between HSCode and Workflow.
type WorkflowTemplateMap struct {
	BaseModel
	HSCodeID             uuid.UUID           `gorm:"type:uuid;column:hs_code_id;not null" json:"hsCodeId"`
	ConsignmentFlow      ConsignmentFlow     `gorm:"type:varchar(50);column:consignment_flow;not null" json:"consignmentFlow"` // e.g., IMPORT, EXPORT
	WorkflowTemplateID   *uuid.UUID          `gorm:"type:uuid;column:workflow_template_id" json:"workflowTemplateId,omitempty"`
	GoWorkflowTemplateID *uuid.UUID          `gorm:"type:uuid;column:go_workflow_template_id" json:"goWorkflowTemplateId,omitempty"`

	// Relationships
	HSCode             HSCode             `gorm:"foreignKey:HSCodeID;references:ID" json:"hsCode"`
	WorkflowTemplate   WorkflowTemplate   `gorm:"foreignKey:WorkflowTemplateID;references:ID" json:"workflowTemplate"`
	GoWorkflowTemplate GoWorkflowTemplate `gorm:"foreignKey:GoWorkflowTemplateID;references:ID" json:"goWorkflowTemplate"`
}

func (w *WorkflowTemplateMap) TableName() string {
	return "workflow_template_maps"
}
