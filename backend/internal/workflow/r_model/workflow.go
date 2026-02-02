package r_model

import "github.com/google/uuid"

type WorkflowTemplate struct {
	BaseModel
	Name          string      `gorm:"type:varchar(100);column:name;not null" json:"name"`      // Name of the workflow template
	Description   string      `gorm:"type:text;column:description" json:"description"`         // Description of the workflow template
	Version       string      `gorm:"type:varchar(50);column:version;not null" json:"version"` // Version of the workflow template
	NodeTemplates []uuid.UUID `gorm:"type:uuid[];column:nodes;not null" json:"nodes"`          // Array of workflow node template IDs
}

func (wt *WorkflowTemplate) TableName() string {
	return "workflow_templates"
}

func (wt *WorkflowTemplate) GetNodeTemplateIDs() []uuid.UUID {
	return wt.NodeTemplates
}
