package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskManager handles task execution and status management
// Architecture: Trader Portal → Workflow Engine → ExecutionUnit Manager
// - Workflow Engine triggers ExecutionUnit Manager to get task info (e.g., form schema)
// - ExecutionUnit Manager executes tasks and determines the next tasks to activate
// - ExecutionUnit Manager notifies Workflow Engine on task completion via Go channel
type TaskManager interface {
	// RegisterTask initializes and executes a task using the provided TaskContext.
	// TaskManager does not have direct access to the Tasks table, so Workflow Manager
	// must provide the TaskContext with the ExecutionUnit already loaded.
	RegisterTask(ctx context.Context, payload InitPayload) (*ExecutionResult, error)

	// HandleExecuteTask is an HTTP handler for executing a task via POST request
	HandleExecuteTask(w http.ResponseWriter, r *http.Request)
}

type ExecutionPayload struct {
	Action  string      `json:"action"`
	Content interface{} `json:"content,omitempty"`
}

// ExecuteTaskRequest represents the request body for task execution
type ExecuteTaskRequest struct {
	ConsignmentID uuid.UUID         `json:"consignment_id"`
	TaskID        uuid.UUID         `json:"task_id"`
	Payload       *ExecutionPayload `json:"payload,omitempty"`
}

// ExecuteTaskResponse represents the response for task execution
type ExecuteTaskResponse struct {
	Success bool             `json:"success"`
	Result  *ExecutionResult `json:"result,omitempty"`
	Error   string           `json:"error,omitempty"`
}

type taskManager struct {
	factory        TaskFactory
	store          *TaskStore                              // Storage for task executions
	completionChan chan<- model.TaskCompletionNotification // Channel to notify Workflow Manager of task completions
	config         *config.Config                          // Application configuration
}

// NewTaskManager creates a new TaskManager instance with persistence data store.
// db is the shared database connection
// completionChan is a channel for notifying Workflow Manager when tasks complete.
func NewTaskManager(db *gorm.DB, completionChan chan<- model.TaskCompletionNotification, cfg *config.Config, formService form.FormService) (TaskManager, error) {
	store, err := NewTaskStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	return &taskManager{
		factory:        NewTaskFactory(cfg, formService),
		store:          store,
		completionChan: completionChan,
		config:         cfg,
	}, nil
}

// HandleExecuteTask is an HTTP handler for executing a task via POST request
func (tm *taskManager) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.TaskID == uuid.Nil {
		writeJSONError(w, http.StatusBadRequest, "task_id is required")
		return
	}
	if req.ConsignmentID == uuid.Nil {
		writeJSONError(w, http.StatusBadRequest, "consignment_id is required")
		return
	}

	// Get task from the store
	ctx := r.Context()
	activeTask, err := tm.getTask(ctx, req.TaskID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("task %s not found: %v", req.TaskID, err))
		return
	}

	// Execute task
	result, err := tm.execute(ctx, activeTask, req.Payload)
	if err != nil {
		slog.ErrorContext(ctx, "failed to execute task",
			"taskID", req.TaskID,
			"consignmentID", req.ConsignmentID,
			"error", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to execute task: "+err.Error())
		return
	}

	// Return success response
	writeJSONResponse(w, http.StatusOK, ExecuteTaskResponse{
		Success: true,
		Result:  result,
	})
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, ExecuteTaskResponse{
		Success: false,
		Error:   message,
	})
}

// RegisterTask initializes and executes a task using the provided TaskContext.
// TaskManager does not have direct access to the Tasks table, so Workflow Manager
// must provide the TaskContext with the ExecutionUnit already loaded.
func (tm *taskManager) RegisterTask(ctx context.Context, payload InitPayload) (*ExecutionResult, error) {

	if payload.GlobalContext == nil {
		payload.GlobalContext = make(map[string]interface{})
	}

	// append the taskId, consignmentId and StepId to the globalContext
	payload.GlobalContext["taskId"] = payload.TaskID
	payload.GlobalContext["consignmentId"] = payload.ConsignmentID

	// Build the executor from the factory
	executor, err := tm.factory.BuildExecutor(ctx, payload.Type, payload.CommandSet, payload.GlobalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build executor: %w", err)
	}

	activeTask := NewActiveTask(payload, executor)

	// Create a task info record
	execution := &TaskInfo{
		ID:            activeTask.TaskID,
		ConsignmentID: payload.ConsignmentID,
		StepID:        payload.StepID,
		Type:          payload.Type,
		Status:        payload.Status,
		CommandSet:    payload.CommandSet,
	}

	globalContextJSON, err := json.Marshal(payload.GlobalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal global context: %w", err)
	}
	execution.GlobalContext = globalContextJSON

	// Store in SQLite
	if err := tm.store.Create(execution); err != nil {
		return nil, fmt.Errorf("failed to store task info: %w", err)
	}

	// Execute a task and return a result to Workflow Manager
	return tm.execute(ctx, activeTask, nil)
}

// execute is a unified method that executes a task and returns the result.
func (tm *taskManager) execute(ctx context.Context, activeTask *ActiveTask, payload *ExecutionPayload) (*ExecutionResult, error) {
	// Check if a task can be executed
	if !activeTask.IsExecutable() {
		return nil, fmt.Errorf("task %s is not ready for execution", activeTask.TaskID)
	}

	// Execute task
	result, err := activeTask.Execute(ctx, payload)
	if err != nil {
		return nil, err
	}

	// Update task status in database
	if err := tm.store.UpdateStatus(activeTask.TaskID, result.Status); err != nil {
		slog.ErrorContext(ctx, "failed to update task status in database",
			"taskID", activeTask.TaskID,
			"error", err)
	}

	if result.Status != "" {
		// Update in-memory status
		activeTask.Status = result.Status
		tm.notifyWorkflowManager(ctx, activeTask.TaskID, result.Status, result.GlobalContextData)
	}

	return result, nil
}

// getTask retrieves a task from the store and combines it with the in-memory executor
func (tm *taskManager) getTask(ctx context.Context, taskID uuid.UUID) (*ActiveTask, error) {
	// Get executions for this task
	// Get executor from the memory cache
	//tm.executorsMu.RLock()
	//executor, exists := tm.executors[taskID]
	//tm.executorsMu.RUnlock()

	execution, err := tm.store.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	// Unmarshal GlobalContext
	var globalContext map[string]interface{}

	if err := json.Unmarshal(execution.GlobalContext, &globalContext); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global context: %w", err)
	}

	// Rebuild executor
	executor, err := tm.factory.BuildExecutor(ctx, execution.Type, execution.CommandSet, globalContext)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild executor: %w", err)
	}

	return &ActiveTask{
		TaskID:        execution.ID,
		ConsignmentID: execution.ConsignmentID,
		StepID:        execution.StepID,
		Type:          execution.Type,
		Status:        execution.Status,
		Executor:      executor,
	}, nil
}

// notifyWorkflowManager sends notification to Workflow Manager via Go channel
func (tm *taskManager) notifyWorkflowManager(ctx context.Context, taskID uuid.UUID, state model.TaskStatus, globalContext map[string]interface{}) {
	if tm.completionChan == nil {
		slog.WarnContext(ctx, "completion channel not configured, skipping notification",
			"taskID", taskID,
			"state", state)
		return
	}

	notification := model.TaskCompletionNotification{
		TaskID:              taskID,
		State:               state,
		AppendGlobalContext: globalContext,
	}

	// Non-blocking send - if a channel is full, log warning but don't block
	select {
	case tm.completionChan <- notification:
		slog.DebugContext(ctx, "task completion notification sent via channel",
			"taskID", taskID,
			"state", state)
	default:
		// Channel is full or closed
		slog.WarnContext(ctx, "completion channel full or unavailable, notification dropped",
			"taskID", taskID,
			"state", state)
	}
}
