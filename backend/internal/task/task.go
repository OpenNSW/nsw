package task

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusSuspended TaskStatus = "SUSPENDED"
	TaskStatusCompleted TaskStatus = "COMPLETED"
	TaskStatusFailed    TaskStatus = "FAILED"
)

// StateManager interface for getting/setting state (Internal and Global)
type StateManager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	GetAll() map[string]interface{}
}

// TaskPluginReturnValue struct for plugin results
type TaskPluginReturnValue struct {
	Status                 TaskStatus
	StatusHumanReadableStr string
	Data                   interface{}
}

// TaskPlugin is the interface that all task types must implement.
type TaskPlugin interface {
	Start(ctx context.Context, config json.RawMessage, is StateManager, gs StateManager) (*TaskPluginReturnValue, error)
	// data is any information provided by callbacks, etc when the task gets resumed
	Resume(ctx context.Context, is StateManager, gs StateManager, data map[string]interface{}) (*TaskPluginReturnValue, error)
}

// TaskContainer is created by TaskManager and loads the plugin. Abstraction for an instance of a Task
type TaskContainer struct {
	TaskID        uuid.UUID
	ConsignmentID uuid.UUID
	Status        TaskStatus
	InternalState StateManager
	GlobalState   StateManager
	Plugin        TaskPlugin
	Config        json.RawMessage
}

func (tc *TaskContainer) Start(ctx context.Context) (*TaskPluginReturnValue, error) {
	return tc.Plugin.Start(ctx, tc.Config, tc.InternalState, tc.GlobalState)
}

func (tc *TaskContainer) Resume(ctx context.Context, data map[string]interface{}) (*TaskPluginReturnValue, error) {
	return tc.Plugin.Resume(ctx, tc.InternalState, tc.GlobalState, data)
}

// InitPayload represents the data required to initialize a task in the system.
type InitPayload struct {
	StepID        string          `json:"stepId" binding:"required"`        // Unique identifier of the step within the workflow template
	TaskID        uuid.UUID       `json:"taskId" binding:"required"`        // Unique identifier of the task instance
	ConsignmentID uuid.UUID       `json:"consignmentId" binding:"required"` // Unique identifier of the instance of a workflow template
	Type          string          `json:"type" binding:"required"`          // Type of the task
	Status        string          `json:"status" binding:"required"`        // Current status of the task
	Config        json.RawMessage `json:"config" binding:"required"`        // Configuration specific to the task
	GlobalContext map[string]interface{}
}
