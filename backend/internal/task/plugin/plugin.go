package plugin

import (
	"context"

	"github.com/google/uuid"
)

type TaskInfo struct {
	Type       Type
	State      State
	TaskID     uuid.UUID
	WorkflowID uuid.UUID
}

// API will be implemented by the TaskContainer, which provides controlled access to
// generic resources and owns all state transitions via the container-level FSM.
type API interface {
	GetTaskID() uuid.UUID
	GetWorkflowID() uuid.UUID
	GetTaskState() State
	ReadFromGlobalStore(key string) (any, bool)
	WriteToLocalStore(key string, value any) error
	ReadFromLocalStore(key string) (any, error)
	GetPluginState() string
	// CanTransition reports whether action is a legal FSM transition from the current plugin state.
	CanTransition(action string) bool
	// Transition applies the FSM transition for action, updating and persisting both
	// plugin state and task state. Returns an error if the action is not permitted.
	Transition(action string) error
}

type ExecutionRequest struct {
	Action  string      `json:"action"`
	Content interface{} `json:"content,omitempty"`
}

type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ApiResponse represents the outcome to be returned to the API caller
type ApiResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`  // Additional data specific to the task type
	Error   *ApiError `json:"error,omitempty"` // Error details if execution failed
}

type GetRenderInfoResponse struct {
	Type        Type   `json:"type"`
	PluginState string `json:"pluginState"`
	State       State  `json:"state"`
	Content     any    `json:"content"`
}

type ExecutionResponse struct {
	NewState            *State
	ExtendedState       *string
	AppendGlobalContext map[string]any
	EmittedOutcome      *string
	Message             string
	ApiResponse         *ApiResponse
}

type Plugin interface {
	Init(api API)
	Start(ctx context.Context) (*ExecutionResponse, error)
	GetRenderInfo(ctx context.Context) (*ApiResponse, error)
	Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error)
}
