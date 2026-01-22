package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskManager handles task execution and status management
// Architecture: Trader Portal → Workflow Engine → Task Manager
// - Workflow Engine triggers Task Manager to get task info (e.g., form schema)
// - Task Manager executes tasks and determines next tasks to activate
// - Task Manager notifies Workflow Engine on task completion via Go channel
// - Workflow Engine uses Task Manager API for task database operations
type TaskManager interface {
	// InitTask initializes and executes a task using the provided TaskContext.
	// TaskManager does not have direct access to Tasks table, so Workflow Manager
	// must provide the TaskContext with the Task already loaded.
	InitTask(ctx context.Context, taskCtx *TaskContext) (*TaskResult, error)

	// UpdateTaskStatus updates the status of a task
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error

	// CreateTask creates a single task
	CreateTask(ctx context.Context, task *model.Task) error

	// UpdateTask updates a single task
	UpdateTask(ctx context.Context, task *model.Task) error

	// OnTaskCompleted is called when a task completes (for async workflows)
	OnTaskCompleted(ctx context.Context, taskID uuid.UUID, result *TaskResult) error

	// SubmitTaskCompletion handles task completion submission from Trader Portal.
	// Validates form data, saves submission, updates task status, and notifies Workflow Manager
	SubmitTaskCompletion(ctx context.Context, taskID uuid.UUID, formData map[string]interface{}) (*TaskResult, error)

	// NotifyTaskCompletion handles task completion notification from external systems (e.g., OGA)
	// Updates task status and notifies Workflow Manager with final state (COMPLETED/REJECTED)
	NotifyTaskCompletion(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error
}

// TaskCompletionNotification represents a notification sent to Workflow Manager when a task completes
type TaskCompletionNotification struct {
	TaskID uuid.UUID
	State  string // Workflow state: "INPROGRESS", "COMPLETED", "REJECTED"
}

type taskManager struct {
	db                   *gorm.DB
	factory              TaskFactory
	completionChan       chan<- TaskCompletionNotification // Channel to notify Workflow Manager of task completions
	workflowEngineClient WorkflowEngineClient              // In-process client for workflow metadata queries (not REST)
	activeTasks          map[uuid.UUID]*model.Task         // In-memory cache of active tasks for fast lookup
	activeTasksMu        sync.RWMutex                      // Mutex for thread-safe access to activeTasks
}

// NewTaskManager creates a new TaskManager instance
// completionChan is a channel for notifying Workflow Manager when tasks complete.
// workflowEngineClient is an in-process client for querying workflow metadata (not REST).
func NewTaskManager(db *gorm.DB, factory TaskFactory, completionChan chan<- TaskCompletionNotification, workflowEngineClient WorkflowEngineClient) TaskManager {
	return &taskManager{
		db:                   db,
		factory:              factory,
		completionChan:       completionChan,
		workflowEngineClient: workflowEngineClient,
		activeTasks:          make(map[uuid.UUID]*model.Task),
	}
}

// InitTask initializes and executes a task using the provided TaskContext.
// TaskManager does not have direct access to Tasks table, so Workflow Manager
// must provide the TaskContext with the Task already loaded.
func (tm *taskManager) InitTask(ctx context.Context, taskCtx *TaskContext) (*TaskResult, error) {
	if taskCtx == nil || taskCtx.Task == nil {
		return nil, fmt.Errorf("task context and task must be provided")
	}

	taskModel := taskCtx.Task

	// Execute task and return result to Workflow Manager
	return tm.execute(ctx, taskModel, nil)
}

// executeTaskInTx executes a task within an existing transaction.
// It handles the common execution flow: create task instance, check CanExecute, execute, update status.
// formData is optional and only used for form task submissions.
func (tm *taskManager) executeTaskInTx(ctx context.Context, tx *gorm.DB, taskModel *model.Task, formData map[string]interface{}) (*TaskResult, error) {
	// 1. Create task instance from factory
	task, err := tm.factory.CreateTask(TaskType(taskModel.Type), taskModel)
	if err != nil {
		return nil, err
	}

	// 2. Build task context with transaction handle
	taskCtx := &TaskContext{
		Task:          taskModel,
		ConsignmentID: taskModel.ConsignmentID,
		AssigneeID:    taskModel.TraderID,
		Tx:            tx,
	}

	// 3. Check if task can execute
	canExecute, err := task.CanExecute(ctx, taskCtx)
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

	// 6. Update task status in database within the transaction
	if err := tx.Model(&model.Task{}).Where("id = ?", taskModel.ID).Update("status", result.Status).Error; err != nil {
		return nil, err
	}

	return result, nil
}

// execute is a unified method that executes a task and returns the result.
// It wraps executeTaskInTx in a transaction and handles Workflow Manager notifications.
// formData is optional and only used for form task submissions.
// If formData is nil, it notifies Workflow Manager with INPROGRESS state.
func (tm *taskManager) execute(ctx context.Context, taskModel *model.Task, formData map[string]interface{}) (*TaskResult, error) {
	var result *TaskResult

	err := tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = tm.executeTaskInTx(ctx, tx, taskModel, formData)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Notify Workflow Manager with INPROGRESS state for tasks without form data
	// (Tasks with form data handle notification in SubmitTaskCompletion after submission)
	if formData == nil {
		tm.notifyWorkflowManager(ctx, taskModel.ID, string(model.WorkflowStateInProgress))
	}

	return result, nil
}

// SubmitTaskCompletion handles task completion submission from Trader Portal.
func (tm *taskManager) SubmitTaskCompletion(ctx context.Context, taskID uuid.UUID, formData map[string]interface{}) (*TaskResult, error) {
	var result *TaskResult

	err := tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Get task from cache or load from database
		taskModel, err := tm.getActiveTask(ctx, taskID)
		if err != nil {
			return err
		}

		// 2. Execute task with submitted form data using unified execution method
		result, err = tm.executeTaskInTx(ctx, tx, taskModel, formData)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 3. Determine workflow state and async notify Workflow Manager
	var workflowState string
	if result.Status == model.TaskStatusApproved || result.Status == model.TaskStatusSubmitted {
		workflowState = string(model.WorkflowStateCompleted)
	} else if result.Status == model.TaskStatusRejected {
		workflowState = string(model.WorkflowStateRejected)
	}

	if workflowState != "" {
		tm.notifyWorkflowManager(ctx, taskID, workflowState)
	}

	return result, nil
}

// NotifyTaskCompletion handles task completion notification from external systems (e.g., OGA)
func (tm *taskManager) NotifyTaskCompletion(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error {
	// Update task status
	if err := tm.UpdateTaskStatus(ctx, taskID, status); err != nil {
		return err
	}

	// Determine workflow state based on status
	var workflowState string
	if status == model.TaskStatusApproved {
		workflowState = string(model.WorkflowStateCompleted)
	} else if status == model.TaskStatusRejected {
		workflowState = string(model.WorkflowStateRejected)
	} else {
		// For other statuses, don't notify
		return nil
	}

	// Notify Workflow Manager
	tm.notifyWorkflowManager(ctx, taskID, workflowState)

	return nil
}

func (tm *taskManager) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status model.TaskStatus) error {
	return tm.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", taskID).Update("status", status).Error
}

func (tm *taskManager) CreateTask(ctx context.Context, task *model.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if err := tm.db.WithContext(ctx).Create(task).Error; err != nil {
		return err
	}

	// Add to activeTasks cache
	tm.activeTasksMu.Lock()
	tm.activeTasks[task.ID] = task
	tm.activeTasksMu.Unlock()

	return nil
}

func (tm *taskManager) UpdateTask(ctx context.Context, task *model.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if err := tm.db.WithContext(ctx).Save(task).Error; err != nil {
		return err
	}

	// Update activeTasks cache
	tm.activeTasksMu.Lock()
	tm.activeTasks[task.ID] = task
	tm.activeTasksMu.Unlock()

	return nil
}

func (tm *taskManager) OnTaskCompleted(ctx context.Context, taskID uuid.UUID, result *TaskResult) error {
	// Update task status
	if err := tm.UpdateTaskStatus(ctx, taskID, result.Status); err != nil {
		return err
	}

	// Determine workflow state based on task result status
	var workflowState string
	if result.Status == model.TaskStatusApproved {
		workflowState = string(model.WorkflowStateCompleted)
	} else if result.Status == model.TaskStatusRejected {
		workflowState = string(model.WorkflowStateRejected)
	} else {
		// For other statuses, don't notify (task is still in progress)
		return nil
	}

	// Notify Workflow Manager via channel
	tm.notifyWorkflowManager(ctx, taskID, workflowState)

	return nil
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

// getActiveTask retrieves a task from the activeTasks cache, or loads from database if not cached
func (tm *taskManager) getActiveTask(ctx context.Context, taskID uuid.UUID) (*model.Task, error) {
	// Try cache first
	tm.activeTasksMu.RLock()
	if task, exists := tm.activeTasks[taskID]; exists {
		tm.activeTasksMu.RUnlock()
		return task, nil
	}
	tm.activeTasksMu.RUnlock()

	// Not in cache, load from database
	taskModel, err := tm.getTaskModelByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Add to cache
	tm.activeTasksMu.Lock()
	tm.activeTasks[taskID] = taskModel
	tm.activeTasksMu.Unlock()

	return taskModel, nil
}

// getTaskModelByID loads a task from the database with all necessary relationships preloaded
func (tm *taskManager) getTaskModelByID(ctx context.Context, taskID uuid.UUID) (*model.Task, error) {
	return tm.getTaskModelByIDWithTx(ctx, tm.db, taskID)
}

// getTaskModelByIDWithTx loads a task from the database with all necessary relationships preloaded
// using the provided transaction handle
func (tm *taskManager) getTaskModelByIDWithTx(ctx context.Context, tx *gorm.DB, taskID uuid.UUID) (*model.Task, error) {
	var taskModel model.Task
	if err := tx.WithContext(ctx).
		Preload("TraderFormTemplate").
		Preload("OGAOfficerFormTemplate").
		First(&taskModel, "id = ?", taskID).Error; err != nil {
		return nil, err
	}
	return &taskModel, nil
}
