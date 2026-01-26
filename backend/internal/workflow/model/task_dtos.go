package model

import (
	"github.com/google/uuid"
)

// TaskCompletionNotification represents a notification sent to Workflow Manager when a task completes
type TaskCompletionNotification struct {
	TaskID uuid.UUID  `json:"taskId" binding:"required"`
	State  TaskStatus `json:"state" binding:"required"`
}
