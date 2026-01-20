package model

import "github.com/google/uuid"

// Task represents a task assigned to a user within a consignment workflow.

type Task struct {
	BaseModel
	Trader   UserInfo   `gorm:"type:jsonb;column:trader;not null" json:"trader"`
	Assignee UserInfo   `gorm:"type:jsonb;column:assignee;not null" json:"assignee"`
	Status   TaskStatus `gorm:"type:varchar(20);column:status;not null" json:"status"` // Status of the task (e.g., PENDING, APPROVED, REJECTED)
}

type UserInfo struct {
	ID               uuid.UUID `gorm:"type:uuid;column:id;not null" json:"userId"`
	FormTemplateID   uuid.UUID `gorm:"type:uuid;column:form_template_id;not null" json:"formTemplateId"`
	FormSubmissionID uuid.UUID `gorm:"type:uuid;column:form_submission_id;not null" json:"formSubmissionId"`
}
