package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// FormSubmission represents a submitted form with data from trader and/or OGA officer
type FormSubmission struct {
	BaseModel
	TaskID        uuid.UUID       `gorm:"type:uuid;column:task_id;not null;index" json:"taskId"`                     // Reference to the task
	ConsignmentID uuid.UUID       `gorm:"type:uuid;column:consignment_id;not null;index" json:"consignmentId"`       // Reference to the consignment
	FormID        uuid.UUID       `gorm:"type:uuid;column:form_id;not null;index" json:"formId"`             // Form identifier (UUID reference to Form.ID)
	FormVersion   string          `gorm:"type:varchar(50);column:form_version;not null" json:"formVersion"`          // Version of the form when submitted
	SubmittedBy   string          `gorm:"type:varchar(255);column:submitted_by;not null" json:"submittedBy"`         // "TRADER" or "OGA"
	FormData      json.RawMessage `gorm:"type:jsonb;column:form_data;not null" json:"formData"`                      // Complete form data (trader + OGA data merged)
	Status        string          `gorm:"type:varchar(50);column:status;not null;default:'SUBMITTED'" json:"status"` // SUBMITTED, APPROVED, REJECTED
}

func (FormSubmission) TableName() string {
	return "form_submissions"
}
