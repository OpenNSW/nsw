package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskManager handles task execution and status management
type TaskManager interface {
	// RegisterPlugin registers a plugin factory for a task type
	RegisterPlugin(taskType string, factory PluginFactory)
	InitTaskContainer(ctx context.Context, payload InitPayload) (*TaskContainer, error)
	NotifyState(ctx context.Context, taskID uuid.UUID, state TaskStatus, globalContext map[string]interface{})
	// HandleExecuteTask is an HTTP handler for executing a task via POST request
	HandleExecuteTask(w http.ResponseWriter, r *http.Request)
	// Close closes the task manager and releases resources
	Close() error
}


type taskManager struct {
	factories      map[string]PluginFactory
	pluginsMu      sync.RWMutex
	store          *TaskStore                   // SQLite storage for task executions
	containers     map[string]*TaskContainer    // In-memory cache for active containers (TaskID string key)
	containersMu   sync.RWMutex
	completionChan chan<- model.TaskCompletionNotification // Channel to notify Workflow Manager of task completions
	config         *config.Config                          // Application configuration
	formService    form.FormService 
}

// NewTaskManager creates a new TaskManager instance with persistence data store.
// db is the shared database connection
// completionChan is a channel for notifying Workflow Manager when tasks complete.
func NewTaskManager(db *gorm.DB, completionChan chan<- model.TaskCompletionNotification, cfg *config.Config, formService form.FormService) (TaskManager, error) {
	store, err := NewTaskStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	tm := &taskManager{
		factories:      make(map[string]PluginFactory),
		store:          store,
		containers:     make(map[string]*TaskContainer),
		completionChan: completionChan,
		config:         cfg,
		formService:    formService,
	}

	// Register default plugins
	tm.RegisterPlugin(TaskTypeSimpleForm, func(api PluginAPI) TaskPlugin {
		// NewSimpleFormTask now returns TaskPlugin directly
		// We assume we will modify NewSimpleFormTask signature
		return NewSimpleFormTask(cfg, api)
	})
	tm.RegisterPlugin(TaskTypeWaitForEvent, func(api PluginAPI) TaskPlugin {
		return &WaitForEventTask{}
	})

	return tm, nil
}

func (tm *taskManager) RegisterPlugin(taskType string, factory PluginFactory) {
	tm.pluginsMu.Lock()
	defer tm.pluginsMu.Unlock()
	tm.factories[taskType] = factory
}

func (tm *taskManager) InitTaskContainer(ctx context.Context, payload InitPayload) (*TaskContainer, error) {
	tm.pluginsMu.RLock()
	factory, exists := tm.factories[payload.Type]
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

	// Init a task container but do NOT execute it yet
	container := &TaskContainer{
		TaskID:           payload.TaskID.String(),
		ConsignmentID:    payload.ConsignmentID,
		Status:           TaskStatus(payload.Status),
		InternalState:    internalState,
		GlobalState:      globalState,
		Config:           payload.Config,
		ExecutionTimeout: time.Duration(tm.config.Tasks.ExecutionTimeoutSeconds) * time.Second,
		FormService:      tm.formService, // Inject Service for API
	}
	
	// Create Plugin via Factory, injecting the Container as API
	container.Plugin = factory(container)

	// Save to store
	internalStateJSON, err := json.Marshal(internalState.GetAll())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal internal state: %w", err)
	}
	globalStateJSON, err := json.Marshal(globalState.GetAll())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal global state: %w", err)
	}

	record := &TaskInfo{
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
		return nil, fmt.Errorf("failed to store task info: %w", err)
	}

	tm.containersMu.Lock()
	tm.containers[payload.TaskID.String()] = container
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
	case TaskStatusAwaitingInput:
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
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	tm.containersMu.RLock()
	container, exists := tm.containers[req.TaskID.String()]
	tm.containersMu.RUnlock()

	if !exists {
		// Try to load from store if not in memory
		record, err := tm.store.GetByID(req.TaskID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "task not found")
			return
		}

		tm.pluginsMu.RLock()
		factory, factoryExists := tm.factories[record.Type]
		tm.pluginsMu.RUnlock()

		if !factoryExists {
			writeJSONError(w, http.StatusInternalServerError, "plugin factory not found")
			return
		}

		var isMap, gsMap map[string]interface{}
		if err := json.Unmarshal(record.InternalState, &isMap); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to unmarshal internal state")
			return
		}
		if err := json.Unmarshal(record.GlobalContext, &gsMap); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to unmarshal global context")
			return
		}
		container = &TaskContainer{
			TaskID:           record.ID.String(),
			ConsignmentID:    record.ConsignmentID,
			Status:           TaskStatus(record.Status),
			InternalState:    NewSimpleMapStateManager(isMap),
			GlobalState:      NewSimpleMapStateManager(gsMap),
			Config:           record.CommandSet,
			ExecutionTimeout: time.Duration(tm.config.Tasks.ExecutionTimeoutSeconds) * time.Second,
			FormService:      tm.formService,
		}
		
		// Hydrate Plugin
		container.Plugin = factory(container)

		// Cache the newly loaded container for future requests
		tm.containersMu.Lock()
		tm.containers[req.TaskID.String()] = container
		tm.containersMu.Unlock()
	}


	status, resultData, err := container.ProcessResume(r.Context(), req.Data)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update status and notify
	// We only have new Status.
	container.Status = status
	taskUUID, err := uuid.Parse(container.TaskID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid task ID format")
		return
	}
	tm.store.UpdateStatus(taskUUID, toModelStatus(status))
	tm.NotifyState(r.Context(), taskUUID, status, container.GlobalState.GetAll())

    // Return backwards-compatible response structure using TaskPluginReturnValue
    response := TaskPluginReturnValue{
        Status: status,
        StatusHumanReadableStr: string(status),
        Data: resultData,
    }

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to write JSON response", "error", err)
	}
}

func (tm *taskManager) Close() error {
	// TODO: Implement graceful shutdown, e.g., closing the task store if needed.
	return nil
}

func toModelStatus(state TaskStatus) model.TaskStatus {
	switch state {
	case TaskStatusCompleted:
		return model.TaskStatusCompleted
	case TaskStatusFailed:
		return model.TaskStatusRejected
	case TaskStatusAwaitingInput:
		return model.TaskStatusInProgress
	default:
		return model.TaskStatusInProgress
	}
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("failed to write json error response", "error", err)
	}
}