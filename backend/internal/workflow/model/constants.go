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
	ConsignmentStateInProgress ConsignmentState = "IN_PROGRESS"
	ConsignmentStateFinished   ConsignmentState = "FINISHED"
)

// TaskStatus represents the status of a task within a workflow.
type TaskStatus string

const (
	TaskStatusLocked    TaskStatus = "LOCKED"    // Task is locked and cannot be worked on because previous tasks are incomplete
	TaskStatusReady     TaskStatus = "READY"     // Task is ready to be worked on
	TaskStatusSubmitted TaskStatus = "SUBMITTED" // Task has been submitted for review
	TaskStatusApproved  TaskStatus = "APPROVED"  // Task has been approved
	TaskStatusRejected  TaskStatus = "REJECTED"  // Task has been rejected and needs rework
)

// FormType represents the type of form within a workflow.
type FormType string

const (
	FormTypeTrader     FormType = "TRADER"      // Form filled only by traders to submit information required to procure permit for HS code
	FormTypeOGAOfficer FormType = "OGA_OFFICER" // Form filled only by OGA officers to submit decision and issue permit
)

// FormSubmisiionStatus represents the status of a form submission.
type FormSubmissionStatus string

const (
	FormSubmissionStatusPending  FormSubmissionStatus = "PENDING"  // Form submission is pending review
	FormSubmissionStatusApproved FormSubmissionStatus = "APPROVED" // Form submission has been approved
	FormSubmissionStatusRejected FormSubmissionStatus = "REJECTED" // Form submission has been rejected
)

// WorkflowState represents the state of a task in the workflow from Workflow Manager's perspective
type WorkflowState string

const (
	WorkflowStateInProgress WorkflowState = "INPROGRESS" // Task is in progress (for non-realtime tasks)
	WorkflowStateCompleted  WorkflowState = "COMPLETED"  // Task has been completed
	WorkflowStateRejected   WorkflowState = "REJECTED"   // Task has been rejected
)
