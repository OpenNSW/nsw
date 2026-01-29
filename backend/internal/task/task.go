package task

import (
	"context"
	"encoding/json"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusSuspended TaskStatus = "SUSPENDED"
	TaskStatusCompleted TaskStatus = "COMPLETED"
	TaskStatusFailed    TaskStatus = "FAILED"
)

// InitPayload represents the data required to initialize a task.
type InitPayload struct {
	StepID        string           `json:"stepId" binding:"required"`        // Unique identifier of the step within the workflow template
	TaskID        uuid.UUID        `json:"taskId" binding:"required"`        // Unique identifier of the task instance
	ConsignmentID uuid.UUID        `json:"consignmentId" binding:"required"` // Unique identifier of the instance of a workflow template
	Type          Type             `json:"type" binding:"required"`          // Type of the task
	Status        model.TaskStatus `json:"status" binding:"required"`        // Current status of the task
	CommandSet    json.RawMessage  `json:"config" binding:"required"`        // Configuration specific to the task
	GlobalContext map[string]interface{}
}

// StateManager deals with state persistence.
type StateManager interface {
	Get(key string) any
	Set(key string, value any)
	GetAll() map[string]any
}

// TaskContainer is created by TaskManager and loads the plugin. Abstraction for an instance of a Task
type TaskContainer struct {
	TaskID        string
	Status        TaskStatus
	InternalState StateManager
	GlobalState   StateManager
}

type TaskPluginReturnValue struct {
	Status                 TaskStatus
	StatusHumanReadableStr string
	Data                   any
}

type TaskPlugin interface {
	Start(ctx context.Context, config map[string]any, is StateManager, gs StateManager) (*TaskPluginReturnValue, error)
	// data is any information provided by callbacks, etc when the task gets resumed
	Resume(ctx context.Context, is StateManager, gs StateManager, data map[string]any) (*TaskPluginReturnValue, error)
}
