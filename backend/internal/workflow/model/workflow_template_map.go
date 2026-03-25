package model

// WorkflowTemplateMap represents the mapping between HSCode and Workflow.
type WorkflowTemplateMap struct {
	BaseModel
	HSCodeID           string          `gorm:"type:text;column:hs_code_id;not null" json:"hsCodeId"`
	ConsignmentFlow    ConsignmentFlow `gorm:"type:varchar(50);column:consignment_flow;not null" json:"consignmentFlow"` // e.g., IMPORT, EXPORT
	WorkflowTemplateID string          `gorm:"type:text;column:workflow_template_id;not null" json:"workflowTemplateId"`

	// Relationships
	HSCode           HSCode           `gorm:"foreignKey:HSCodeID;references:ID" json:"hsCode"`
	WorkflowTemplate WorkflowTemplate `gorm:"foreignKey:WorkflowTemplateID;references:ID" json:"workflowTemplate"`
}

func (w *WorkflowTemplateMap) TableName() string {
	return "workflow_template_maps"
}

type WorkflowTemplateMapV2 struct {
	BaseModel
	HSCodeID           string          `gorm:"type:text;column:hs_code_id;not null" json:"hsCodeId"`
	ConsignmentFlow    ConsignmentFlow `gorm:"type:varchar(50);column:consignment_flow;not null" json:"consignmentFlow"` // e.g., IMPORT, EXPORT
	WorkflowTemplateID string          `gorm:"type:text;column:workflow_template_id;not null" json:"workflowTemplateId"`

	// Relationships
	HSCode           HSCode             `gorm:"foreignKey:HSCodeID;references:ID" json:"hsCode"`
	WorkflowTemplate WorkflowTemplateV2 `gorm:"foreignKey:WorkflowTemplateID;references:ID" json:"workflowTemplate"`
}

func (w *WorkflowTemplateMapV2) TableName() string {
	return "workflow_template_maps_v2"
}
