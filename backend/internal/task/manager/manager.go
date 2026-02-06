package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/task/container"
	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
)

type InitTaskRequest struct {
	ConsignmentID uuid.UUID   `json:"consignment_id"`
	TaskID        uuid.UUID   `json:"task_id"`
	StepID        string      `json:"step_id"`
	Type          plugin.Type `json:"type"`
	GlobalState   map[string]any
	Config        json.RawMessage `json:"config"`
}

type InitTaskResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ExecuteTaskResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// TaskManager handles task execution and status management
// Architecture: Trader Portal → Workflow Engine → Task Manager
// - Workflow Manager triggers Task Manager to get task info (e.g., form schema)
// - ExecutionUnit Manager executes tasks and determines the next tasks to activate
// - ExecutionUnit Manager notifies Workflow Engine on task completion via Go channel
type TaskManager interface {
	// InitTask initializes and executes a task using the provided TaskContext.
	InitTask(ctx context.Context, request InitTaskRequest) (*InitTaskResponse, error)

	// HandleExecuteTask is an HTTP handler for executing a task via POST request
	HandleExecuteTask(w http.ResponseWriter, r *http.Request)
}

// ExecuteTaskRequest represents the request body for task execution
type ExecuteTaskRequest struct {
	ConsignmentID uuid.UUID                `json:"consignment_id"`
	TaskID        uuid.UUID                `json:"task_id"`
	Payload       *plugin.ExecutionRequest `json:"payload,omitempty"`
}

type taskManager struct {
	factory          plugin.TaskFactory
	store            persistence.TaskStoreInterface     // Storage for task executions
	completionChan   chan<- WorkflowManagerNotification // Channel to notify Workflow Manager of task completions
	config           *config.Config                     // Application configuration
	containerCache   *containerCache                    // LRU cache for active containers
	containerBuildMu sync.Mutex                         // Protects container creation to prevent duplicates
}

// NewTaskManager creates a new TaskManager instance with persistence data store.
// db is the shared database connection
// completionChan is a channel for notifying Workflow Manager when tasks complete.
// Note: The completionChan should have a sufficient buffer size (recommended: 1000+)
// to prevent notification drops during high load.
func NewTaskManager(db *gorm.DB, completionChan chan<- WorkflowManagerNotification, cfg *config.Config, formService form.FormService) (TaskManager, error) {
	store, err := persistence.NewTaskStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	// Initialize container cache with capacity of 100 active containers
	cache := newContainerCache(100)

	return &taskManager{
		factory:        plugin.NewTaskFactory(cfg, formService),
		store:          store,
		completionChan: completionChan,
		config:         cfg,
		containerCache: cache,
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
	writeJSONResponse(w, http.StatusOK, result.ApiResponse)
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

// InitTask initializes a new task container, creates its execution record,
// and starts the task. It builds the plugin executor, sets up local state management,
// creates a container with the executor and state managers, persists the task record
// to the database, and invokes the plugin's Start method.
// Returns InitTaskResponse on success, or an error if initialization or start fails.
func (tm *taskManager) InitTask(ctx context.Context, request InitTaskRequest) (*InitTaskResponse, error) {
	// Check if container already exists in cache
	if existing, found := tm.containerCache.Get(request.TaskID); found {
		slog.WarnContext(ctx, "task already initialized, reusing existing container",
			"taskID", request.TaskID)
		return tm.start(ctx, existing)
	}

	// Build the executor from the factory
	executor, err := tm.factory.BuildExecutor(ctx, request.Type, request.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to build executor: %w", err)
	}

	// Generate the state manager
	localStateManager, err := persistence.NewLocalStateManager(
		tm.store,
		request.TaskID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create local state manager: %w", err)
	}

	// Defensive copy of GlobalState to prevent external modifications causing race conditions
	globalStateCopy := make(map[string]any, len(request.GlobalState))
	for k, v := range request.GlobalState {
		globalStateCopy[k] = v
	}

	activeTask := container.NewContainer(request.TaskID, request.ConsignmentID, request.StepID, globalStateCopy, localStateManager, tm.store, executor)

	// Convert request.Config to json.RawMessage
	configBytes, err := json.Marshal(request.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task config: %w", err)
	}

	globalContextBytes, err := json.Marshal(request.GlobalState)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal global context: %w", err)
	}

	// Create a task execution record
	taskInfo := &persistence.TaskInfo{
		ID:            activeTask.TaskID,
		ConsignmentID: request.ConsignmentID,
		StepID:        request.StepID,
		Type:          request.Type,
		State:         plugin.InProgress,
		Config:        configBytes,
		GlobalContext: globalContextBytes,
	}

	// Store in SQLite
	if err := tm.store.Create(taskInfo); err != nil {
		return nil, fmt.Errorf("failed to store task info: %w", err)
	}

	// Cache the active container
	tm.containerCache.Set(request.TaskID, activeTask)
	slog.InfoContext(ctx, "container added to cache",
		"taskID", request.TaskID,
		"cacheSize", tm.containerCache.Len())

	// Execute a task and return a result to Workflow Manager
	return tm.start(ctx, activeTask)
}

func (tm *taskManager) start(ctx context.Context, activeTask *container.Container) (*InitTaskResponse, error) {
	_, err := activeTask.Start(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	return &InitTaskResponse{Success: true}, nil
}

// execute is a unified method that executes a task and returns the result.
func (tm *taskManager) execute(ctx context.Context, activeTask *container.Container, payload *plugin.ExecutionRequest) (*plugin.ExecutionResponse, error) {
	// Execute task
	result, err := activeTask.Execute(ctx, payload)
	if err != nil {
		return nil, err
	}

	taskId := activeTask.GetTaskID()
	// Update task status in database

	if result.NewState != nil {
		// Update persistent status
		if err := tm.store.UpdateStatus(taskId, result.NewState); err != nil {
			slog.ErrorContext(ctx, "failed to update task status in database",
				"taskID", taskId,
				"error", err)
		}
		tm.notifyWorkflowManager(ctx, activeTask.TaskID, result.NewState)
	}

	return result, nil
}

// getTask retrieves a task from the cache or store and combines it with the in-memory executor and returns a task container.
// Uses double-checked locking to prevent duplicate container creation.
func (tm *taskManager) getTask(ctx context.Context, taskID uuid.UUID) (*container.Container, error) {
	// First check (unlocked, fast path for cache hits)
	if cachedContainer, found := tm.containerCache.Get(taskID); found {
		slog.DebugContext(ctx, "container retrieved from cache",
			"taskID", taskID)
		return cachedContainer, nil
	}

	// Lock to prevent duplicate container creation
	tm.containerBuildMu.Lock()
	defer tm.containerBuildMu.Unlock()

	// Second check after acquiring lock (another goroutine may have created it)
	if cachedContainer, found := tm.containerCache.Get(taskID); found {
		slog.DebugContext(ctx, "container retrieved from cache after lock",
			"taskID", taskID)
		return cachedContainer, nil
	}

	// Cache miss - rebuild from persistence
	slog.DebugContext(ctx, "container not in cache, rebuilding from persistence",
		"taskID", taskID)

	execution, err := tm.store.GetByID(taskID)
	if err != nil {
		return nil, err
	}

	taskConfig := json.RawMessage{}

	// Only unmarshal if Config is not empty
	if len(execution.Config) > 0 {
		err = json.Unmarshal(execution.Config, &taskConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Rebuild executor
	executor, err := tm.factory.BuildExecutor(ctx, execution.Type, taskConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild executor: %w", err)
	}

	localState, err := persistence.NewLocalStateManagerWithCache(
		tm.store,
		execution.ID,
		execution.LocalState,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create local state manager: %w", err)
	}

	globalContext := map[string]any{}

	// Only unmarshal if GlobalContext is not empty
	if len(execution.GlobalContext) > 0 {
		err = json.Unmarshal(execution.GlobalContext, &globalContext)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal global context: %w", err)
		}
	}

	activeContainer := container.NewContainer(
		execution.ID, execution.ConsignmentID, execution.StepID, globalContext, localState, tm.store, executor)

	// Cache the rebuilt container
	tm.containerCache.Set(taskID, activeContainer)
	slog.InfoContext(ctx, "container rebuilt and cached",
		"taskID", taskID,
		"cacheSize", tm.containerCache.Len())

	return activeContainer, nil
}

// notifyWorkflowManager sends notification to Workflow Manager via Go channel
func (tm *taskManager) notifyWorkflowManager(ctx context.Context, taskID uuid.UUID, state *plugin.State) {
	if tm.completionChan == nil {
		slog.WarnContext(ctx, "completion channel not configured, skipping notification",
			"taskID", taskID,
			"state", state)
		return
	}

	notification := WorkflowManagerNotification{
		TaskID:       taskID,
		UpdatedState: state,
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
