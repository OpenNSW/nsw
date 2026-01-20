package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// FormTemplate represents the template of a form in the particular step of a consignment workflow.
type FormTemplate struct {
	BaseModel
	FormType FormType        `gorm:"type:varchar(50);column:form_type;not null" json:"formType"` // e.g., TRADER, OGA_OFFICER
	Schema   json.RawMessage `gorm:"type:jsonb;column:schema;not null" json:"schema"`            // Store the form template as JSONB
	UISchema json.RawMessage `gorm:"type:jsonb;column:ui_schema;not null" json:"uiSchema"`       // Store the UI schema as JSONB
}

func (f *FormTemplate) TableName() string {
	return "form_templates"
}

// FormSubmission represents the submission of a form within a consignment workflow.
type FormSubmission struct {
	BaseModel
	FormTemplateID uuid.UUID            `gorm:"type:uuid;column:form_template_id;not null" json:"formTemplateId"` // Reference to the Form Template
	ConsignmentID  uuid.UUID            `gorm:"type:uuid;column:consignment_id;not null" json:"consignmentId"`    // Reference to the Consignment
	Data           json.RawMessage      `gorm:"type:jsonb;column:data;not null" json:"data"`                      // Store the submitted form data as JSONB (e.g., JSON)
	Status         FormSubmissionStatus `gorm:"type:varchar(20);column:status;not null" json:"status"`            // Status of the submission (e.g., PENDING, APPROVED, REJECTED)

	// Relationships
	Consignment  Consignment  `gorm:"foreignKey:ConsignmentID;references:ID" json:"consignment"`
	FormTemplate FormTemplate `gorm:"foreignKey:FormTemplateID;references:ID" json:"formTemplate"`
}

func (f *FormSubmission) TableName() string {
	return "form_submissions"
}
