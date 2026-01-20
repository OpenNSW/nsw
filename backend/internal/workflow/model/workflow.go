package model

// WorkflowTemplate represents the template of a workflow for consignments.
type WorkflowTemplate struct {
	BaseModel
	Template string `gorm:"type:text;not null" json:"template"` // Store the template in XML
}

func (w *WorkflowTemplate) TableName() string {
	return "workflow_templates"
}
