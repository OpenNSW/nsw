package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// TaskManager handles task execution and status management
// Architecture: Trader Portal → Workflow Engine → Task Manager
// - Workflow Engine triggers Task Manager to get task info (e.g., form schema)
// - Task Manager executes tasks and determines next tasks to activate
// - Task Manager notifies Workflow Engine on task completion via Go channel
type TaskManager interface {
	// InitTask initializes and executes a task using the provided TaskContext.
	// TaskManager does not have direct access to Tasks table, so Workflow Manager
	// must provide the TaskContext with the Task already loaded.
	InitTask(ctx context.Context, taskCtx *TaskContext) (*TaskResult, error)

	// SubmitTaskCompletion handles task completion submission from Trader Portal.
	// Validates form data, saves submission, updates task status, and notifies Workflow Manager
	SubmitTaskCompletion(ctx context.Context, taskID uuid.UUID, formData map[string]interface{}) (*TaskResult, error)

	// RegisterTask registers a task in the active tasks cache
	RegisterTask(task *model.Task)

	// GetTask retrieves a task from the active tasks cache
	GetTask(taskID uuid.UUID) (*model.Task, bool)

	// UpdateTaskStatus updates the status of a task in memory
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error

	// HandleExecuteTask is an HTTP handler for executing a task via POST request
	HandleExecuteTask(w http.ResponseWriter, r *http.Request)
}

// TaskCompletionNotification represents a notification sent to Workflow Manager when a task completes
type TaskCompletionNotification struct {
	TaskID uuid.UUID
	State  string // Workflow state: "IN_PROGRESS", "COMPLETED", "REJECTED"
}

// ExecuteTaskRequest represents the request body for task execution
type ExecuteTaskRequest struct {
	ConsignmentID uuid.UUID              `json:"consignment_id"`
	TaskID        uuid.UUID              `json:"task_id"`
	FormData      map[string]interface{} `json:"form_data,omitempty"`
}

// ExecuteTaskResponse represents the response for task execution
type ExecuteTaskResponse struct {
	Success bool        `json:"success"`
	Result  *TaskResult `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type taskManager struct {
	factory        TaskFactory
	completionChan chan<- TaskCompletionNotification // Channel to notify Workflow Manager of task completions
	activeTasks    map[uuid.UUID]*model.Task         // In-memory cache of active tasks for fast lookup
	activeTasksMu  sync.RWMutex                      // Mutex for thread-safe access to activeTasks
}

// NewTaskManager creates a new TaskManager instance
// completionChan is a channel for notifying Workflow Manager when tasks complete.
func NewTaskManager(completionChan chan<- TaskCompletionNotification) TaskManager {
	return &taskManager{
		factory:        NewTaskFactory(),
		activeTasks:    make(map[uuid.UUID]*model.Task),
		completionChan: completionChan,
	}
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

	// Get task from cache
	taskModel, exists := tm.GetTask(req.TaskID)
	if !exists {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("task %s not found", req.TaskID))
		return
	}

	// Verify consignment ID matches
	if taskModel.ConsignmentID != req.ConsignmentID {
		writeJSONError(w, http.StatusBadRequest, "consignment_id does not match task")
		return
	}

	// Execute task
	ctx := r.Context()
	result, err := tm.execute(ctx, taskModel, req.FormData)
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
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, ExecuteTaskResponse{
		Success: false,
		Error:   message,
	})
}

// InitTask initializes and executes a task using the provided TaskContext.
// TaskManager does not have direct access to Tasks table, so Workflow Manager
// must provide the TaskContext with the Task already loaded.
func (tm *taskManager) InitTask(ctx context.Context, taskCtx *TaskContext) (*TaskResult, error) {
	if taskCtx == nil || taskCtx.Task == nil {
		return nil, fmt.Errorf("task context and task must be provided")
	}

	taskModel := taskCtx.Task

	// Register task in active tasks cache
	tm.RegisterTask(taskModel)

	// Execute task and return result to Workflow Manager
	return tm.execute(ctx, taskModel, nil)
}

// execute is a unified method that executes a task and returns the result.
// formData is optional and only used for form task submissions.
// If formData is nil, it notifies Workflow Manager with INPROGRESS state.
func (tm *taskManager) execute(ctx context.Context, taskModel *model.Task, formData map[string]interface{}) (*TaskResult, error) {
	// 1. Create task instance from factory
	task, err := tm.factory.CreateTask(TaskType(taskModel.Type), taskModel)
	if err != nil {
		return nil, err
	}

	// 2. Build task context
	taskCtx := &TaskContext{
		Task:          taskModel,
		ConsignmentID: taskModel.ConsignmentID,
	}

	// 3. Check if task can execute
	canExecute, err := task.CanExecute()
	if err != nil {
		return nil, err
	}
	if !canExecute {
		return nil, fmt.Errorf("task %s is not ready for execution", taskModel.ID)
	}

	// 4. Prepare context with form data if provided (for form task submissions)
	execCtx := ctx
	if formData != nil {
		execCtx = context.WithValue(ctx, "formData", formData)
	}

	// 5. Execute task
	result, err := task.Execute(execCtx, taskCtx)
	if err != nil {
		return nil, err
	}

	// 6. Update task status in memory
	taskModel.Status = result.Status

	// Notify Workflow Manager with IN_PROGRESS state for tasks without form data
	// (Tasks with form data handle notification in SubmitTaskCompletion after submission)
	if formData == nil {
		tm.notifyWorkflowManager(ctx, taskModel.ID, string(model.TaskStatusInProgress))
	}

	return result, nil
}

// SubmitTaskCompletion handles task completion submission from Trader Portal.
func (tm *taskManager) SubmitTaskCompletion(ctx context.Context, taskID uuid.UUID, formData map[string]interface{}) (*TaskResult, error) {
	// 1. Get a task from a cache
	taskModel, exists := tm.GetTask(taskID)
	if !exists {
		return nil, fmt.Errorf("task %s not found in active tasks", taskID)
	}

	// 2. Execute a task with submitted form data
	result, err := tm.execute(ctx, taskModel, formData)
	if err != nil {
		return nil, err
	}

	tm.notifyWorkflowManager(ctx, taskID, string(result.Status))

	return result, nil
}

func (tm *taskManager) UpdateTaskStatus(_ context.Context, taskID uuid.UUID, status model.TaskStatus) error {
	tm.activeTasksMu.Lock()
	defer tm.activeTasksMu.Unlock()

	task, exists := tm.activeTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found in active tasks", taskID)
	}

	task.Status = status
	return nil
}

// RegisterTask registers a task in the active tasks cache
func (tm *taskManager) RegisterTask(task *model.Task) {
	if task == nil {
		return
	}

	tm.activeTasksMu.Lock()
	tm.activeTasks[task.ID] = task
	tm.activeTasksMu.Unlock()
}

// GetTask retrieves a task from the active tasks cache
func (tm *taskManager) GetTask(taskID uuid.UUID) (*model.Task, bool) {
	tm.activeTasksMu.RLock()
	defer tm.activeTasksMu.RUnlock()

	task, exists := tm.activeTasks[taskID]
	return task, exists
}

// notifyWorkflowManager sends notification to Workflow Manager via Go channel
func (tm *taskManager) notifyWorkflowManager(ctx context.Context, taskID uuid.UUID, state string) {
	if tm.completionChan == nil {
		slog.WarnContext(ctx, "completion channel not configured, skipping notification",
			"taskID", taskID,
			"state", state)
		return
	}

	notification := TaskCompletionNotification{
		TaskID: taskID,
		State:  state,
	}

	// Non-blocking send - if channel is full, log warning but don't block
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
