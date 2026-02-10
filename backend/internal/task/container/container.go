package container

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
)

type Container struct {
	TaskID           uuid.UUID
	ConsignmentID    *uuid.UUID
	PreConsignmentID *uuid.UUID
	StepID           string
	State            plugin.State
	Executable       plugin.Plugin
	globalState      map[string]any
	localState       persistence.Manager
	taskStore        persistence.TaskStoreInterface
	pluginState      string // Cache for plugin-level business state
	mu               sync.RWMutex
}

func (c *Container) GetTaskState() plugin.State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

func (c *Container) SetTaskState(state plugin.State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = state
}

func (c *Container) Init(api plugin.API) {
	c.Executable.Init(api)
}

func (c *Container) Start(ctx context.Context) (*plugin.ExecutionResponse, error) {
	return c.Executable.Start(ctx)
}

func (c *Container) Execute(ctx context.Context, request *plugin.ExecutionRequest) (*plugin.ExecutionResponse, error) {
	return c.Executable.Execute(ctx, request)
}

func (c *Container) GetTaskID() uuid.UUID {
	return c.TaskID
}

func (c *Container) GetConsignmentID() *uuid.UUID {
	return c.ConsignmentID
}

func (c *Container) WriteToLocalStore(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.localState.SetState(key, value)
}

func (c *Container) ReadFromLocalStore(key string) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.localState.GetState(key)
}

func (c *Container) ReadFromGlobalStore(key string) (any, bool) {
	// check whether the key exists
	if _, ok := c.globalState[key]; !ok {
		return nil, false
	}

	return c.globalState[key], true
}

func (c *Container) GetPluginState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pluginState
}

func (c *Container) SetPluginState(state string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pluginState = state
	// Persist to database
	return c.taskStore.UpdatePluginState(c.TaskID, state)
}

// NewContainer creates a new container for a task with a given Executable plugin
func NewContainer(taskId uuid.UUID, consignmentId *uuid.UUID, preConsignmentId *uuid.UUID, stepId string, globalStore map[string]any, localStore persistence.Manager, taskStore persistence.TaskStoreInterface, executable plugin.Plugin) *Container {
	c := &Container{
		TaskID:           taskId,
		ConsignmentID:    consignmentId,
		PreConsignmentID: preConsignmentId,
		StepID:           stepId,
		Executable:       executable,
		globalState:      globalStore,
		localState:       localStore,
		taskStore:        taskStore,
	}

	// Load plugin state from database
	if taskStore != nil {
		pluginState, err := taskStore.GetPluginState(taskId)
		if err == nil {
			c.pluginState = pluginState
		}
		// If error, pluginState remains empty string (default)
	}

	executable.Init(c)

	return c
}
