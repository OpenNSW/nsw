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

// API will be implemented by the TaskContainer, which provides controlled access to Generic Resources
type API interface {
	GetTaskID() uuid.UUID
	GetWorkflowID() uuid.UUID
	GetTaskState() State
	SetTaskState(state State)
	ReadFromGlobalStore(key string) (any, bool)
	WriteToLocalStore(key string, value any) error
	ReadFromLocalStore(key string) (any, error)
	GetPluginState() string
	SetPluginState(state string) error
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
	Message             string
	ApiResponse         *ApiResponse
}

type Plugin interface {
	Init(api API)
	Start(ctx context.Context) (*ExecutionResponse, error)
	GetRenderInfo(ctx context.Context) (*ApiResponse, error)
	Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error)
}
