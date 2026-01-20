package model

// ConsignmentType represents the type of consignment.
type ConsignmentType string

const (
	ConsignmentTypeImport ConsignmentType = "IMPORT"
	ConsignmentTypeExport ConsignmentType = "EXPORT"
)

// ConsignmentState represents the state of a consignment in the workflow.
type ConsignmentState string

const (
	StateInProgress ConsignmentState = "IN_PROGRESS"
	StateFinished   ConsignmentState = "FINISHED"
)

// TaskStatus represents the status of a task within a workflow.
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "PENDING"
	TaskStatusApproved TaskStatus = "APPROVED"
	TaskStatusRejected TaskStatus = "REJECTED"
)

// FormType represents the type of form within a workflow.
type FormType string

const (
	FormTypeTrader     FormType = "TRADER"      // Form filled only by traders to submit information required to procure permit for HS code
	FormTypeOGAOfficer FormType = "OGA_OFFICER" // Form filled only by OGA officers to submit decision and issue permit
)
