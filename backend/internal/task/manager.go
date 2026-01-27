package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// TaskFetcher is a callback function to fetch task details from workflow database
// Used when a task is not found in the execution database
type TaskFetcher func(ctx context.Context, taskID uuid.UUID) (*InitPayload, error)

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

	// SetTaskFetcher sets a callback to fetch tasks from workflow database when not found in execution database
	SetTaskFetcher(fetcher TaskFetcher)

	// HandleExecuteTask is an HTTP handler for executing a task via POST request
	HandleExecuteTask(w http.ResponseWriter, r *http.Request)

	// GetTask retrieves a task execution record by its ID
	GetTask(taskID uuid.UUID) (*TaskRecord, error)

	// GetTasksByConsignment retrieves task execution records by consignment ID
	GetTasksByConsignment(consignmentID uuid.UUID) ([]TaskRecord, error)

	// Close closes the task manager and releases resources
	Close() error
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
	factory       TaskFactory
	store         *TaskStore                  // SQLite storage for task executions
	executors     map[uuid.UUID]ExecutionUnit // In-memory cache for executors (can't be serialized)
	//executorsMu    sync.RWMutex                            // Mutex for thread-safe access to executors
	completionChan chan<- model.TaskCompletionNotification // Channel to notify Workflow Manager of task completions
	taskFetcher    TaskFetcher                             // Optional callback to fetch tasks from workflow database
}

// NewTaskManager creates a new TaskManager instance with SQLite persistence
// dbPath is the path to the SQLite database file (use ":memory:" for an in-memory database)
// completionChan is a channel for notifying Workflow Manager when tasks complete.
func NewTaskManager(dbPath string, completionChan chan<- model.TaskCompletionNotification) (TaskManager, error) {
	store, err := NewTaskStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	return &taskManager{
		factory:        NewTaskFactory(),
		store:          store,
		//executors:      make(map[uuid.UUID]ExecutionUnit),
		completionChan: completionChan,
		taskFetcher:    nil, // Will be set by Workflow Manager if needed
	}, nil
}

// NewTaskManagerWithStore creates a TaskManager with a provided store (useful for testing)
func NewTaskManagerWithStore(store *TaskStore, completionChan chan<- model.TaskCompletionNotification) TaskManager {
	return &taskManager{
		factory:        NewTaskFactory(),
		store:          store,
		executors:      make(map[uuid.UUID]ExecutionUnit),
		completionChan: completionChan,
		taskFetcher:    nil,
	}
}

// Close closes the task manager and releases resources
func (tm *taskManager) Close() error {
	if tm.store != nil {
		return tm.store.Close()
	}
	return nil
}

// HandleExecuteTask is an HTTP handler for executing a task via POST request
// Route: POST /api/tasks/{taskId}
func (tm *taskManager) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract taskId from URL path
	taskIDStr := r.PathValue("taskId")
	if taskIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "taskId is required in path")
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid taskId format")
		return
	}

	// Parse request body for payload (optional)
	var reqBody struct {
		Payload *ExecutionPayload `json:"payload,omitempty"`
	}
	// Body might be empty, so we don't fail if decode fails
	json.NewDecoder(r.Body).Decode(&reqBody)

	// Get task from the store
	activeTask, err := tm.getTask(taskID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("task %s not found: %v", taskID, err))
		return
	}

	// Execute task
	ctx := r.Context()
	result, err := tm.execute(ctx, activeTask, reqBody.Payload)
	if err != nil {
		slog.ErrorContext(ctx, "failed to execute task",
			"taskID", taskID,
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
	// Build the executor from the factory
	executor, err := tm.factory.BuildExecutor(payload.Type, payload.CommandSet)
	if err != nil {
		return nil, fmt.Errorf("failed to build executor: %w", err)
	}

	activeTask := NewActiveTask(payload, executor)

	// Create a task execution record
	execution := &TaskRecord{
		ID:            activeTask.TaskID,
		ConsignmentID: payload.ConsignmentID,
		StepID:        payload.StepID,
		Type:          payload.Type,
		Status:        payload.Status,
		CommandSet:    payload.CommandSet,
	}

	// Store in SQLite
	if err := tm.store.Create(execution); err != nil {
		return nil, fmt.Errorf("failed to store task execution: %w", err)
	}

	// Create an ActiveTask for execution

	// Cache executor in memory
	//tm.executorsMu.Lock()
	//tm.executors[activeTask.TaskID] = activeTask
	//tm.executorsMu.Unlock()

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

	// Update result data in database if provided
	if result.Data != nil {
		dataBytes, err := json.Marshal(result.Data)
		if err != nil {
			slog.ErrorContext(ctx, "failed to marshal result data", "taskID", activeTask.TaskID, "error", err)
		} else if err := tm.store.db.Model(&TaskRecord{}).Where("id = ?", activeTask.TaskID).Update("result_data", dataBytes).Error; err != nil {
			slog.ErrorContext(ctx, "failed to update result data in database",
				"taskID", activeTask.TaskID,
				"error", err)
		}
	}

	// Update in-memory status
	activeTask.Status = result.Status

	// Notify Workflow Manager
	tm.notifyWorkflowManager(ctx, activeTask.TaskID, result.Status)

	return result, nil
}

// GetTask retrieves a task execution record by its ID via the store
func (tm *taskManager) GetTask(taskID uuid.UUID) (*TaskRecord, error) {
	return tm.store.GetByID(taskID)
}

// GetTasksByConsignment retrieves task execution records by consignment ID via the store
func (tm *taskManager) GetTasksByConsignment(consignmentID uuid.UUID) ([]TaskRecord, error) {
	return tm.store.GetByConsignmentID(consignmentID)
}

// SetTaskFetcher sets a callback to fetch tasks from workflow database when not found in execution database
func (tm *taskManager) SetTaskFetcher(fetcher TaskFetcher) {
	tm.taskFetcher = fetcher
}

// getTask retrieves a task from the store and combines it with the in-memory executor
// If task is not found and a taskFetcher is set, it will try to fetch from workflow database and auto-register
func (tm *taskManager) getTask(taskID uuid.UUID) (*ActiveTask, error) {
	execution, err := tm.store.GetByID(taskID)
	if err == nil {
		// Task found in execution database, rebuild executor and return
		executor, err := tm.factory.BuildExecutor(execution.Type, execution.CommandSet)
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

	// Task not found in execution database, try to fetch from workflow database if fetcher is set
	if tm.taskFetcher != nil {
		ctx := context.Background()
		payload, fetchErr := tm.taskFetcher(ctx, taskID)
		if fetchErr == nil && payload != nil {
			// Auto-register the task
			slog.InfoContext(ctx, "auto-registering task from workflow database",
				"taskID", taskID)
			_, regErr := tm.RegisterTask(ctx, *payload)
			if regErr != nil {
				slog.WarnContext(ctx, "failed to auto-register task",
					"taskID", taskID,
					"error", regErr)
				// Continue to try getting it again
			} else {
				// Try getting it again after registration
				execution, err = tm.store.GetByID(taskID)
				if err == nil {
					executor, err := tm.factory.BuildExecutor(execution.Type, execution.CommandSet)
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
			}
		}
	}

	// Return original error if we couldn't fetch or register
	return nil, err
}

// notifyWorkflowManager sends notification to Workflow Manager via Go channel
func (tm *taskManager) notifyWorkflowManager(ctx context.Context, taskID uuid.UUID, state model.TaskStatus) {
	if tm.completionChan == nil {
		slog.WarnContext(ctx, "completion channel not configured, skipping notification",
			"taskID", taskID,
			"state", state)
		return
	}

	notification := model.TaskCompletionNotification{
		TaskID: taskID,
		State:  state,
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
