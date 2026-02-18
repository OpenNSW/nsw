package container

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
)

type Container struct {
	TaskID                 uuid.UUID
	WorkflowID             uuid.UUID
	WorkflowNodeTemplateID uuid.UUID
	State                  plugin.State
	Executable             plugin.Plugin
	globalState            map[string]any
	localState             persistence.Manager
	taskStore              persistence.TaskStoreInterface
	pluginState            string // Cache for plugin-level business state
	fsm                    *plugin.PluginFSM
	mu                     sync.RWMutex
}

func (c *Container) GetTaskState() plugin.State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

func (c *Container) Init(api plugin.API) {
	c.Executable.Init(api)
}

// CanTransition reports whether action is a legal FSM transition from the current plugin state.
func (c *Container) CanTransition(action string) bool {
	if c.fsm == nil {
		return true
	}
	return c.fsm.CanTransition(c.GetPluginState(), action)
}

// Transition applies the FSM transition for action, updating in-memory state and
// persisting both plugin state and task state to the store.
func (c *Container) Transition(action string) error {
	if c.fsm == nil {
		return nil
	}
	outcome, err := c.fsm.Transition(c.GetPluginState(), action)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.pluginState = outcome.NextPluginState
	if outcome.NextTaskState != "" {
		c.State = outcome.NextTaskState
	}
	c.mu.Unlock()
	if err := c.taskStore.UpdatePluginState(c.TaskID, outcome.NextPluginState); err != nil {
		return err
	}
	if outcome.NextTaskState != "" {
		return c.taskStore.UpdateStatus(c.TaskID, &outcome.NextTaskState)
	}
	return nil
}

func (c *Container) Start(ctx context.Context) (*plugin.ExecutionResponse, error) {
	prev := c.GetPluginState()
	resp, err := c.Executable.Start(ctx)
	if err != nil {
		return resp, err
	}
	if resp == nil {
		resp = &plugin.ExecutionResponse{}
	}
	c.mu.RLock()
	state, pluginState := c.State, c.pluginState
	c.mu.RUnlock()
	if pluginState != prev {
		resp.NewState = &state
		resp.ExtendedState = &pluginState
	}
	return resp, nil
}

func (c *Container) GetRenderInfo(ctx context.Context) (*plugin.ApiResponse, error) {
	return c.Executable.GetRenderInfo(ctx)
}

func (c *Container) Execute(ctx context.Context, request *plugin.ExecutionRequest) (*plugin.ExecutionResponse, error) {
	prev := c.GetPluginState()
	resp, err := c.Executable.Execute(ctx, request)
	if err != nil {
		return resp, err
	}
	if resp == nil {
		resp = &plugin.ExecutionResponse{}
	}
	c.mu.RLock()
	state, pluginState := c.State, c.pluginState
	c.mu.RUnlock()
	if pluginState != prev {
		resp.NewState = &state
		resp.ExtendedState = &pluginState
	}
	return resp, nil
}

func (c *Container) GetTaskID() uuid.UUID {
	return c.TaskID
}

func (c *Container) GetWorkflowID() uuid.UUID {
	return c.WorkflowID
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

// NewContainer creates a new container for a task with a given Executable plugin and FSM.
// initialState is the task-level state to restore (InProgress for new tasks, or the
// persisted state when rebuilding from the store after a cache miss).
func NewContainer(taskId uuid.UUID, workflowId uuid.UUID, workflowNodeTemplateId uuid.UUID, initialState plugin.State, globalStore map[string]any, localStore persistence.Manager, taskStore persistence.TaskStoreInterface, executable plugin.Plugin, fsm *plugin.PluginFSM) *Container {
	c := &Container{
		TaskID:                 taskId,
		WorkflowID:             workflowId,
		WorkflowNodeTemplateID: workflowNodeTemplateId,
		State:                  initialState,
		Executable:             executable,
		globalState:            globalStore,
		localState:             localStore,
		taskStore:              taskStore,
		fsm:                    fsm,
	}

	if taskStore != nil {
		pluginState, err := taskStore.GetPluginState(taskId)
		if err == nil {
			c.pluginState = pluginState
		}
	}

	executable.Init(c)

	return c
}
