package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// FormTemplate represents the template of a form in the particular step of a consignment workflow.
type FormTemplate struct {
	BaseModel
	FormType FormType `gorm:"type:varchar(50);column:form_type;not null" json:"formType"` // e.g., TRADER, OGA_OFFICER
	Template string   `gorm:"type:text;column:template;not null" json:"template"`         // Store the form template as text
}

// FormSubmission represents the submission of a form within a consignment workflow.
type FormSubmission struct {
	BaseModel
	FormTemplateID uuid.UUID       `gorm:"type:uuid;column:form_template_id;not null" json:"formTemplateId"` // Reference to the Form Template
	ConsignmentID  uuid.UUID       `gorm:"type:uuid;column:consignment_id;not null" json:"consignmentId"`    // Reference to the Consignment
	Data           json.RawMessage `gorm:"type:jsonb;column:data;not null" json:"data"`                      // Store the submitted form data as JSONB (e.g., JSON)
	Status         string          `gorm:"type:varchar(20);column:status;not null" json:"status"`            // Status of the submission (e.g., PENDING, APPROVED, REJECTED)
}
