package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InitTaskInTaskManagerDTO represents the data required to initialize a task in the Task Manager system.
type InitTaskInTaskManagerDTO struct {
	TaskID uuid.UUID       `json:"taskId" binding:"required"` // Unique identifier of the task
	Type   StepType        `json:"type" binding:"required"`   // Type of the task
	Status TaskStatus      `json:"status" binding:"required"` // Current status of the task
	Config json.RawMessage `json:"config" binding:"required"` // Configuration specific to the task
}

// TaskCompletionNotification represents a notification sent to Workflow Manager when a task completes
type TaskCompletionNotification struct {
	TaskID    uuid.UUID  `json:"taskId" binding:"required"`
	State     TaskStatus `json:"state" binding:"required"`
	Timestamp time.Time  `json:"timestamp" binding:"required"`
}
