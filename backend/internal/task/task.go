package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/form/model"
	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusAwaitingInput TaskStatus = "AWAITING_INPUT"
	TaskStatusCompleted     TaskStatus = "COMPLETED"
	TaskStatusFailed        TaskStatus = "FAILED"
)

// StateManager interface (internal use by Container/API)
type StateManager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	GetAll() map[string]interface{}
}

// PluginAPI will be implemented by the TaskContainer and exposed to Plugins
type PluginAPI interface {
	// Form Service Access
	GetFormById(ctx context.Context, formId uuid.UUID) (*model.FormResponse, error)
	
	// Local Storage Access
	WriteToLocalStore(key string, value interface{}) error
	ReadFromLocalStore(key string) (interface{}, bool)

	// Global Storage Access
	WriteToGlobalStore(key string, value interface{}) error
	ReadFromGlobalStore(key string) (interface{}, bool)
}

// TaskPluginReturnValue struct for plugin results
type TaskPluginReturnValue struct {
	Status                 TaskStatus
	StatusHumanReadableStr string
	Data                   interface{}
}

// TaskPlugin is the interface that all task types must implement.
type TaskPlugin interface {
	Start(ctx context.Context, config map[string]interface{}) (*TaskPluginReturnValue, error)
	Resume(ctx context.Context, data map[string]interface{}) (*TaskPluginReturnValue, error)
}

// PluginFactory is a function that creates a new plugin instance given the API
type PluginFactory func(api PluginAPI) TaskPlugin

// TaskContainer is created by TaskManager and loads the plugin. Abstraction for an instance of a Task
// It acts as a Supervisor and implements PluginAPI.
type TaskContainer struct {
	TaskID        string
	ConsignmentID uuid.UUID
	Status        TaskStatus
	InternalState StateManager
	GlobalState   StateManager
	Plugin        TaskPlugin
	Config        json.RawMessage
	ExecutionTimeout time.Duration
	
	// Services required by API
	FormService form.FormService
}

// --- PluginAPI Implementation ---

func (tc *TaskContainer) GetFormById(ctx context.Context, formId uuid.UUID) (*model.FormResponse, error) {
	if tc.FormService == nil {
		return nil, fmt.Errorf("form service not available in container")
	}
	return tc.FormService.GetFormByID(ctx, formId)
}

func (tc *TaskContainer) WriteToLocalStore(key string, value interface{}) error {
	return tc.InternalState.Set(key, value)
}

func (tc *TaskContainer) ReadFromLocalStore(key string) (interface{}, bool) {
	return tc.InternalState.Get(key)
}

func (tc *TaskContainer) WriteToGlobalStore(key string, value interface{}) error {
	return tc.GlobalState.Set(key, value)
}

func (tc *TaskContainer) ReadFromGlobalStore(key string) (interface{}, bool) {
	return tc.GlobalState.Get(key)
}

// --- Supervisor Logic ---
func (tc *TaskContainer) Execute(ctx context.Context, config map[string]any) (status TaskStatus, data any, err error) {
	// SECURITY 1: Panic Recovery
	defer func() {
		if r := recover(); r != nil {
			tc.Status = TaskStatusFailed
			slog.ErrorContext(ctx, "CRITICAL: Task panicked", "taskID", tc.TaskID, "panic", r)
			status = TaskStatusFailed
			data = nil
			err = fmt.Errorf("panic recovered in task %s: %v", tc.TaskID, r)
		}
	}()
	
	// SECURITY 3: Timeouts
	timeout := tc.ExecutionTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute Logic
	result, err := tc.Plugin.Start(ctx, config)
	if err != nil {
		tc.Status = TaskStatusFailed
		return TaskStatusFailed, nil, err
	}

	tc.Status = result.Status
	return result.Status, result.Data, nil
}

func (tc *TaskContainer) ProcessResume(ctx context.Context, data map[string]any) (status TaskStatus, resultData any, err error) {
	// SECURITY 1: Panic Recovery
	defer func() {
		if r := recover(); r != nil {
			tc.Status = TaskStatusFailed
			slog.ErrorContext(ctx, "CRITICAL: Task panicked during Resume", "taskID", tc.TaskID, "panic", r)
			status = TaskStatusFailed
			resultData = nil
			err = fmt.Errorf("panic recovered during resume for task %s: %v", tc.TaskID, r)
		}
	}()

	// SECURITY 3: Timeouts
	timeout := tc.ExecutionTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute Logic
	result, err := tc.Plugin.Resume(ctx, data)
	if err != nil {
		tc.Status = TaskStatusFailed
		return TaskStatusFailed, nil, err
	}

	tc.Status = result.Status
	return result.Status, result.Data, nil
}

// InitPayload represents the data required to initialize a task in the system.
type InitPayload struct {
	StepID        string          `json:"stepId" binding:"required"`
	TaskID        uuid.UUID       `json:"taskId" binding:"required"`
	ConsignmentID uuid.UUID       `json:"consignmentId" binding:"required"`
	Type          string          `json:"type" binding:"required"`
	Status        string          `json:"status" binding:"required"`
	Config        json.RawMessage `json:"config" binding:"required"`
	GlobalContext map[string]interface{}
}
