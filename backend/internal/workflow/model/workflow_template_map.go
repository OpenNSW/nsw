package model

// WorkflowTemplateMap represents the mapping between HSCode and Workflow.
type WorkflowTemplateMap struct {
	BaseModel
	HSCodeID           string `gorm:"type:text;column:hs_code_id;not null" json:"hsCodeId"`
	ConsignmentFlow    string `gorm:"type:varchar(50);column:consignment_flow;not null" json:"consignmentFlow"` // e.g., IMPORT, EXPORT
	WorkflowTemplateID string `gorm:"type:text;column:workflow_template_id;not null" json:"workflowTemplateId"`

	// Relationships
	WorkflowTemplate WorkflowTemplate `gorm:"foreignKey:WorkflowTemplateID;references:ID" json:"workflowTemplate"`
}

func (w *WorkflowTemplateMap) TableName() string {
	return "workflow_template_maps"
}
