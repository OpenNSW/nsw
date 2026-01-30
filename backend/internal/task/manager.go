package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// TaskManager handles task execution and status management
type TaskManager interface {
	RegisterPlugin(taskType string, plugin TaskPlugin)
	InitTaskContainer(ctx context.Context, payload InitPayload) (*TaskContainer, error)
	NotifyState(ctx context.Context, taskID uuid.UUID, state TaskStatus, globalContext map[string]interface{})
	// HandleExecuteTask is an HTTP handler for executing a task via POST request
	HandleExecuteTask(w http.ResponseWriter, r *http.Request)
	// Close closes the task manager and releases resources
	Close() error
}

type taskManager struct {
	plugins        map[string]TaskPlugin
	pluginsMu      sync.RWMutex
	store          *TaskStore                   // SQLite storage for task executions
	containers     map[uuid.UUID]*TaskContainer // In-memory cache for active containers
	containersMu   sync.RWMutex
	completionChan chan<- model.TaskCompletionNotification // Channel to notify Workflow Manager of task completions
	config         *config.Config                          // Application configuration
}

// NewTaskManager creates a new TaskManager instance with SQLite persistence
// dbPath is the path to the SQLite database file (use ":memory:" for an in-memory database)
// completionChan is a channel for notifying Workflow Manager when tasks complete.
func NewTaskManager(dbPath string, completionChan chan<- model.TaskCompletionNotification, cfg *config.Config, formService form.FormService) (TaskManager, error) {
	store, err := NewTaskStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	tm := &taskManager{
		plugins:        make(map[string]TaskPlugin),
		store:          store,
		containers:     make(map[uuid.UUID]*TaskContainer),
		completionChan: completionChan,
		config:         cfg,
	}

	// Register default plugins
	simpleFormPlugin, err := NewSimpleFormTask(cfg, formService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize simple form task: %w", err)
	}
	tm.RegisterPlugin("SIMPLE_FORM", simpleFormPlugin)
	tm.RegisterPlugin("WAIT_FOR_EVENT", &WaitForEventTask{})

	return tm, nil
}

func (tm *taskManager) RegisterPlugin(taskType string, plugin TaskPlugin) {
	tm.pluginsMu.Lock()
	defer tm.pluginsMu.Unlock()
	tm.plugins[taskType] = plugin
}

func (tm *taskManager) InitTaskContainer(ctx context.Context, payload InitPayload) (*TaskContainer, error) {
	tm.pluginsMu.RLock()
	plugin, exists := tm.plugins[payload.Type]
	tm.pluginsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no plugin registered for task type: %s", payload.Type)
	}

	if payload.GlobalContext == nil {
		payload.GlobalContext = make(map[string]interface{})
	}

	// append the taskId, consignmentId and StepId to the globalContext
	payload.GlobalContext["taskId"] = payload.TaskID.String()
	payload.GlobalContext["consignmentId"] = payload.ConsignmentID.String()

	// Create state managers
	internalState := NewSimpleMapStateManager(nil)
	globalState := NewSimpleMapStateManager(payload.GlobalContext)

	container := &TaskContainer{
		TaskID:        payload.TaskID,
		ConsignmentID: payload.ConsignmentID,
		Status:        TaskStatus(payload.Status),
		InternalState: internalState,
		GlobalState:   globalState,
		Plugin:        plugin,
		Config:        payload.Config,
	}

	// Save to store
	internalStateJSON, err := json.Marshal(internalState.GetAll())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal internal state: %w", err)
	}
	globalStateJSON, err := json.Marshal(globalState.GetAll())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal global state: %w", err)
	}

	record := &TaskRecord{
		ID:            payload.TaskID,
		ConsignmentID: payload.ConsignmentID,
		StepID:        payload.StepID,
		Type:          payload.Type,
		Status:        model.TaskStatus(payload.Status),
		CommandSet:    payload.Config,
		InternalState: internalStateJSON,
		GlobalContext: globalStateJSON,
	}

	if err := tm.store.Create(record); err != nil {
		return nil, fmt.Errorf("failed to store task record: %w", err)
	}

	tm.containersMu.Lock()
	tm.containers[payload.TaskID] = container
	tm.containersMu.Unlock()

	return container, nil
}

func (tm *taskManager) NotifyState(ctx context.Context, taskID uuid.UUID, state TaskStatus, globalContext map[string]interface{}) {
	if tm.completionChan == nil {
		return
	}

	// Map TaskStatus to model.TaskStatus for compatibility with WorkflowManager
	var modelStatus model.TaskStatus
	switch state {
	case TaskStatusCompleted:
		modelStatus = model.TaskStatusCompleted
	case TaskStatusFailed:
		modelStatus = model.TaskStatusRejected
	case TaskStatusSuspended:
		modelStatus = model.TaskStatusInProgress
	default:
		modelStatus = model.TaskStatusInProgress
	}

	notification := model.TaskCompletionNotification{
		TaskID:              taskID,
		State:               modelStatus,
		AppendGlobalContext: globalContext,
	}

	select {
	case tm.completionChan <- notification:
	default:
		slog.WarnContext(ctx, "completion channel full, dropping notification", "taskID", taskID)
	}
}

func (tm *taskManager) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	// Implementation for handling external triggers (e.g. form submission)
	// This should call Resume on the appropriate container
	var req struct {
		TaskID uuid.UUID              `json:"task_id"`
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tm.containersMu.RLock()
	container, exists := tm.containers[req.TaskID]
	tm.containersMu.RUnlock()

	if !exists {
		// Try to load from store if not in memory
		record, err := tm.store.GetByID(req.TaskID)
		if err != nil {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}

		tm.pluginsMu.RLock()
		plugin, pluginExists := tm.plugins[record.Type]
		tm.pluginsMu.RUnlock()

		if !pluginExists {
			http.Error(w, "plugin not found", http.StatusInternalServerError)
			return
		}

		var isMap, gsMap map[string]interface{}
		if err := json.Unmarshal(record.InternalState, &isMap); err != nil {
			http.Error(w, "failed to unmarshal internal state", http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(record.GlobalContext, &gsMap); err != nil {
			http.Error(w, "failed to unmarshal global context", http.StatusInternalServerError)
			return
		}
		container = &TaskContainer{
			TaskID:        record.ID,
			ConsignmentID: record.ConsignmentID,
			Status:        TaskStatus(record.Status),
			InternalState: NewSimpleMapStateManager(isMap),
			GlobalState:   NewSimpleMapStateManager(gsMap),
			Plugin:        plugin,
			Config:        record.CommandSet,
		}
	}

	result, err := container.Resume(r.Context(), req.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update status and notify
	container.Status = result.Status
	tm.store.UpdateStatus(container.TaskID, model.TaskStatus(result.Status))
	tm.NotifyState(r.Context(), container.TaskID, result.Status, container.GlobalState.(*SimpleMapStateManager).GetAll())

	json.NewEncoder(w).Encode(result)
}

func (tm *taskManager) Close() error {
	return tm.store.Close()
}
