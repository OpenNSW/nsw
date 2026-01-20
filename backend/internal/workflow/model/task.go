package model

import "github.com/google/uuid"

// Task represents a task assigned to a user within a consignment workflow.

type Task struct {
	BaseModel
	TraderID                   uuid.UUID  `gorm:"type:uuid;column:trader_id;not null" json:"traderId"` // Reference to the Trader
	TraderFormTemplateID       uuid.UUID  `gorm:"type:uuid;column:trader_form_template_id;not null" json:"traderFormTemplateId"`
	TraderFormSubmissionID     *uuid.UUID `gorm:"type:uuid;column:trader_form_submission_id" json:"traderFormSubmissionId,omitempty"`
	OGAOfficerID               uuid.UUID  `gorm:"type:uuid;column:oga_officer_id;not null" json:"ogaOfficerId"` // Reference to the OGA Officer
	OGAOfficerFormTemplateID   uuid.UUID  `gorm:"type:uuid;column:oga_officer_form_template_id;not null" json:"ogaOfficerFormTemplateId"`
	OGAOfficerFormSubmissionID *uuid.UUID `gorm:"type:uuid;column:oga_officer_form_submission_id" json:"ogaOfficerFormSubmissionId,omitempty"`
	ConsignmentID              uuid.UUID  `gorm:"type:uuid;column:consignment_id;not null" json:"consignmentId"` // Reference to the Consignment
	Status                     TaskStatus `gorm:"type:varchar(20);column:status;not null" json:"status"`         // Status of the task (e.g., LOCKED, READY, SUBMITTED, APPROVED, REJECTED)

	// Relationships
	Consignment              Consignment     `gorm:"foreignKey:ConsignmentID;references:ID" json:"consignment"`
	TraderFormTemplate       FormTemplate    `gorm:"foreignKey:TraderFormTemplateID;references:ID" json:"traderFormTemplate"`
	OGAOfficerFormTemplate   FormTemplate    `gorm:"foreignKey:OGAOfficerFormTemplateID;references:ID" json:"ogaOfficerFormTemplate"`
	TraderFormSubmission     *FormSubmission `gorm:"foreignKey:TraderFormSubmissionID;references:ID" json:"traderFormSubmission,omitempty"`
	OGAOfficerFormSubmission *FormSubmission `gorm:"foreignKey:OGAOfficerFormSubmissionID;references:ID" json:"ogaOfficerFormSubmission,omitempty"`
}

func (t *Task) TableName() string {
	return "tasks"
}
