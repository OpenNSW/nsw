package registry

import (
	"encoding/json"
	"sync"

	engine "github.com/OpenNSW/core/workflow"
	"github.com/OpenNSW/core/taskflow/orchestrator"
)

// InMemRegistry is a basic in-memory implementation of orchestrator.TaskTemplateRegistry.
// Reads and writes are guarded by mu so reload-at-runtime callers don't race
// the temporal workers reading templates.
type InMemRegistry struct {
	mu        sync.RWMutex
	tasks     map[string]orchestrator.TaskTemplate
	subtasks  map[string]orchestrator.SubTaskTemplate
	workflows map[string]engine.WorkflowDefinition
	generics  map[string]json.RawMessage
}

func NewInMemRegistry() *InMemRegistry {
	return &InMemRegistry{
		tasks:     make(map[string]orchestrator.TaskTemplate),
		subtasks:  make(map[string]orchestrator.SubTaskTemplate),
		workflows: make(map[string]engine.WorkflowDefinition),
		generics:  make(map[string]json.RawMessage),
	}
}

func (r *InMemRegistry) RegisterTask(t orchestrator.TaskTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t
}

func (r *InMemRegistry) RegisterSubTask(st orchestrator.SubTaskTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subtasks[st.ID] = st
}

func (r *InMemRegistry) RegisterWorkflow(w engine.WorkflowDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[w.ID] = w
}

func (r *InMemRegistry) RegisterGeneric(id string, data json.RawMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generics[id] = data
}

func (r *InMemRegistry) GetTaskTemplate(id string) (orchestrator.TaskTemplate, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	return t, ok
}

func (r *InMemRegistry) GetSubTaskTemplate(id string) (orchestrator.SubTaskTemplate, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	st, ok := r.subtasks[id]
	return st, ok
}

func (r *InMemRegistry) GetWorkflow(id string) (engine.WorkflowDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.workflows[id]
	return w, ok
}

func (r *InMemRegistry) GetGenericTemplate(id string) (json.RawMessage, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.generics[id]
	return g, ok
}
