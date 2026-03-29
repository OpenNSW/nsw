package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/task/container"
	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
)

type InitTaskRequest struct {
	// Task ID is the unique identifier for this task instance.
	TaskID string `json:"task_id"`
	// Workflow ID is the unique identifier for the currently active workflow instance.
	// TODO: This is not used, remove this in a separate PR
	WorkflowID string `json:"workflow_id"`
	// WorkflowNodeTemplateID is the unique identifier for the currently active workflow
	// node template (aka task template).
	// TODO: This is not used, remove this in a separate PR
	WorkflowNodeTemplateID string      `json:"workflow_node_template_id"`
	Type                   plugin.Type `json:"type"`
	GlobalState            map[string]any
	Config                 json.RawMessage `json:"config"`
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

// WorkflowUpdateHandler handles task update notifications for the workflow manager.
type WorkflowUpdateHandler func(ctx context.Context, taskID string, state *plugin.State, extendedState *string, appendGlobalContext map[string]any, outcome *string)

// WorkflowDoneHandler handles task completion notifications for the workflow manager.
// TODO: these functions should return an error?
type WorkflowDoneHandler func(ctx context.Context, workflowID, taskID string, outputs map[string]any)

// TaskManager handles task execution and status management
// Architecture: Trader Portal → Workflow Engine → Task Manager
// - Workflow Manager triggers Task Manager to get task info (e.g., form schema)
// - ExecutionUnit Manager executes tasks and determines the next tasks to activate
// - ExecutionUnit Manager notifies Workflow Engine on task completion via registered update handler
type TaskManager interface {
	// InitTask initializes and executes a task using the provided TaskContext.
	InitTask(ctx context.Context, request InitTaskRequest) (*InitTaskResponse, error)

	// Core Domain Methods
	ExecuteTask(ctx context.Context, req ExecuteTaskRequest) (*plugin.ExecutionResponse, error)
	GetTaskRenderInfo(ctx context.Context, taskID string) (*plugin.ApiResponse, error)

	// Used by Old WorkflowManager
	// RegisterUpstreamCallback registers the callback used for task updates.
	RegisterUpstreamCallback(callback WorkflowUpdateHandler)

	// User by New WorkflowManager V2
	// RegisterUpstreamDoneCallback registers the callback used when task is done.
	RegisterUpstreamDoneCallback(callback WorkflowDoneHandler)
	// RegisterUpstreamUpdateCallback registers the callback used when task state changes.
	RegisterUpstreamUpdateCallback(callback WorkflowUpdateHandler)
}

// ExecuteTaskRequest represents the request body for task execution
type ExecuteTaskRequest struct {
	WorkflowID string                   `json:"workflow_id"`
	TaskID     string                   `json:"task_id"`
	Payload    *plugin.ExecutionRequest `json:"payload,omitempty"`
}

type taskManager struct {
	factory               plugin.TaskFactory
	store                 persistence.TaskStoreInterface // Storage for task executions
	workflowUpdateHandler WorkflowUpdateHandler          // Handler used to notify Workflow Manager of task updates
	workflowDoneHandler   WorkflowDoneHandler            // Handler used to notify Workflow Manager of task completions
	containerCache        *containerCache                // LRU cache for active containers
	containerBuildMu      sync.Mutex                     // Protects container creation to prevent duplicates
	useWorkflowManagerV2  bool                           // Are we using the  workflow manager v2. This is a temporary flag during the migration.
}

// NewTaskManager creates a new TaskManager instance with persistence data store.
func NewTaskManager(db *gorm.DB, factory plugin.TaskFactory) (TaskManager, error) {
	store, err := persistence.NewTaskStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	// Initialize container cache with capacity of 100 active containers
	cache := newContainerCache(100)

	return &taskManager{
		factory:        factory,
		store:          store,
		containerCache: cache,
	}, nil
}

// RegisterUpstreamUpdateCallback registers the callback used for task updates.
// TODO: delete this after the migration to the new WorkflowManager.
func (tm *taskManager) RegisterUpstreamCallback(callback WorkflowUpdateHandler) {
	tm.useWorkflowManagerV2 = false
	tm.workflowUpdateHandler = callback
}

// RegisterUpstreamUpdateCallback registers the callback used for task updates.
func (tm *taskManager) RegisterUpstreamUpdateCallback(callback WorkflowUpdateHandler) {
	tm.useWorkflowManagerV2 = true
	tm.workflowUpdateHandler = callback
}

// RegisterUpstreamDoneCallback registers the callback used for task completion notifications.
func (tm *taskManager) RegisterUpstreamDoneCallback(callback WorkflowDoneHandler) {
	tm.useWorkflowManagerV2 = true
	tm.workflowDoneHandler = callback
}

// GetTaskRenderInfo retrieves task rendering info (core logic)
func (tm *taskManager) GetTaskRenderInfo(ctx context.Context, taskID string) (*plugin.ApiResponse, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required")
	}

	activeTask, err := tm.getTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task %s not found: %w", taskID, err)
	}

	result, err := activeTask.GetRenderInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get render info for task %s: %w", taskID, err)
	}

	return result, nil
}

// ExecuteTask is the core logic for executing a task
func (tm *taskManager) ExecuteTask(ctx context.Context, req ExecuteTaskRequest) (*plugin.ExecutionResponse, error) {
	if req.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	activeTask, err := tm.getTask(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task %s not found: %w", req.TaskID, err)
	}

	result, err := tm.execute(ctx, activeTask, req.Payload)
	if err != nil {
		slog.ErrorContext(ctx, "failed to execute task",
			"taskID", req.TaskID,
			"workflowID", req.WorkflowID,
			"error", err)
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}
	return result, nil
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
	exec, err := tm.factory.BuildExecutor(ctx, request.Type, request.Config)
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

	activeTask := container.NewContainer(request.TaskID, request.WorkflowID, request.WorkflowNodeTemplateID, plugin.Initialized, globalStateCopy, localStateManager, tm.store, exec.Plugin, exec.FSM)

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
		ID:                     activeTask.TaskID,
		WorkflowID:             request.WorkflowID,
		WorkflowNodeTemplateID: request.WorkflowNodeTemplateID,
		Type:                   request.Type,
		State:                  plugin.Initialized,
		Config:                 configBytes,
		GlobalContext:          globalContextBytes,
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
	result, err := activeTask.Start(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	// Notify the workflow manager of the initial state after starting the task (e.g., InProgress). This ensures that
	//the workflow manager is aware of the task's state change immediately after initialization.
	tm.notifyWorkflowUpdateHandler(ctx, activeTask.TaskID, result.NewState, result.ExtendedState, result.AppendGlobalContext, result.EmittedOutcome)

	return &InitTaskResponse{Success: true}, nil
}

// execute is a unified method that executes a task and returns the result.
func (tm *taskManager) execute(ctx context.Context, activeTask *container.Container, payload *plugin.ExecutionRequest) (*plugin.ExecutionResponse, error) {
	// Execute task
	result, err := activeTask.Execute(ctx, payload)
	if err != nil {
		return nil, err
	}

	if result.NewState != nil {
		if tm.useWorkflowManagerV2 {
			if *result.NewState == plugin.Completed || *result.NewState == plugin.Failed {
				tm.notifyWorkflowDoneHandler(ctx, activeTask.WorkflowID, activeTask.TaskID, result.AppendGlobalContext)
			} else {
				tm.notifyWorkflowUpdateHandler(ctx, activeTask.TaskID, result.NewState, result.ExtendedState, result.AppendGlobalContext, result.EmittedOutcome)
			}
		} else {
			tm.notifyWorkflowUpdateHandler(ctx, activeTask.TaskID, result.NewState, result.ExtendedState, result.AppendGlobalContext, result.EmittedOutcome)
		}
	}

	return result, nil
}

// getTask retrieves a task from the cache or store and combines it with the in-memory executor and returns a task container.
// Uses double-checked locking to prevent duplicate container creation.
func (tm *taskManager) getTask(ctx context.Context, taskID string) (*container.Container, error) {
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
	exec, err := tm.factory.BuildExecutor(ctx, execution.Type, taskConfig)
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
		execution.ID, execution.WorkflowID, execution.WorkflowNodeTemplateID, execution.State, globalContext, localState, tm.store, exec.Plugin, exec.FSM)

	// Cache the rebuilt container
	tm.containerCache.Set(taskID, activeContainer)
	slog.InfoContext(ctx, "container rebuilt and cached",
		"taskID", taskID,
		"cacheSize", tm.containerCache.Len())

	return activeContainer, nil
}

// notifyWorkflowUpdateHandler sends state updates to Workflow Manager via the registered handler.
func (tm *taskManager) notifyWorkflowUpdateHandler(ctx context.Context, taskID string, state *plugin.State, extendedState *string, appendGlobalContext map[string]any, outcome *string) {
	if tm.workflowUpdateHandler == nil {
		slog.WarnContext(ctx, "workflow manager callback not configured, skipping notification",
			"taskID", taskID,
			"state", state,
			"extendedState", extendedState,
			"appendGlobalContext", appendGlobalContext,
		)
		return
	}

	tm.workflowUpdateHandler(ctx, taskID, state, extendedState, appendGlobalContext, outcome)
	slog.DebugContext(ctx, "task completion notification sent via callback",
		"taskID", taskID,
		"state", state,
		"extendedState", extendedState,
		"appendGlobalContext", appendGlobalContext,
	)
}

func (tm *taskManager) notifyWorkflowDoneHandler(
	ctx context.Context,
	workflowID string,
	taskID string,
	appendGlobalContext map[string]any) {
	if tm.workflowDoneHandler == nil {
		slog.WarnContext(ctx, "workflow manager callback not configured, skipping notification",
			"taskID", taskID,
			"appendGlobalContext", appendGlobalContext,
		)
		return
	}

	tm.workflowDoneHandler(ctx, workflowID, taskID, appendGlobalContext)
	slog.DebugContext(ctx, "task completion notification sent via callback",
		"taskID", taskID,
		"appendGlobalContext", appendGlobalContext,
	)
}
