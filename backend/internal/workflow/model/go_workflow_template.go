package model

import (
	"encoding/json"
)

// GoWorkflowTemplate represents a workflow definition in the go-workflow format.
type GoWorkflowTemplate struct {
	BaseModel
	Name       string          `gorm:"type:varchar(255);column:name;not null" json:"name"`
	Definition json.RawMessage `gorm:"type:jsonb;column:definition;not null;serializer:json" json:"definition"`
}

func (GoWorkflowTemplate) TableName() string {
	return "go_workflow_templates"
}
