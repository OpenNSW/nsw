package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// TaskType represents the type of a task within a workflow.

// TaskStatus represents the status of a task within a workflow.
type TaskStatus string

const (
	TaskStatusLocked    TaskStatus = "LOCKED"     // Task is locked and cannot be worked on because previous tasks are incomplete
	TaskStatusReady     TaskStatus = "READY"      // Task is ready to be worked on
	TaskStatusSubmitted TaskStatus = "INPROGRESS" // Task is in progress after being submitted by the trader to get some action from OGA officer
	TaskStatusCompleted TaskStatus = "COMPLETED"  // Task has been Completed to proceed with the next steps
	TaskStatusRejected  TaskStatus = "REJECTED"   // Task has been rejected and needs rework
)

// DependencyStatus represents the completion status of a dependency
type DependencyStatus string

const (
	DependencyStatusIncomplete DependencyStatus = "INCOMPLETE" // Dependency is not yet completed
	DependencyStatusCompleted  DependencyStatus = "COMPLETED"  // Dependency has been completed
)

// Task represents a task instance within a consignment workflow.
type Task struct {
	BaseModel
	ConsignmentID uuid.UUID                   `gorm:"type:uuid;column:consignment_id;not null" json:"consignmentId"`          // Reference to the Consignment
	StepID        string                      `gorm:"type:varchar(100);column:step_id;not null" json:"stepId"`                // Step ID from the workflow template
	Type          StepType                    `gorm:"type:varchar(50);column:type;not null" json:"type"`                      // Type of the task (e.g., TRADER_FORM, OGA_FORM)
	Status        TaskStatus                  `gorm:"type:varchar(20);column:status;not null" json:"status"`                  // Status of the task (e.g., LOCKED, READY, SUBMITTED, APPROVED, REJECTED)
	Config        json.RawMessage             `gorm:"type:jsonb;column:config;not null" json:"config"`                        // Configuration specific to the task
	DependsOn     map[string]DependencyStatus `gorm:"type:jsonb;column:depends_on;serializer:json;not null" json:"dependsOn"` // Map of stepID to completion status

	// Relationships
	Consignment Consignment `gorm:"foreignKey:ConsignmentID;references:ID" json:"consignment"`
}

func (t *Task) TableName() string {
	return "tasks"
}
