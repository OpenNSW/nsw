package registry

import (
	"encoding/json"

	engine "github.com/OpenNSW/go-temporal-workflow"
	"github.com/OpenNSW/nsw-task-flow/orchestrator"
)

// InMemRegistry is a basic in-memory implementation of orchestrator.TaskTemplateRegistry.
type InMemRegistry struct {
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
	r.tasks[t.ID] = t
}

func (r *InMemRegistry) RegisterSubTask(st orchestrator.SubTaskTemplate) {
	r.subtasks[st.ID] = st
}

func (r *InMemRegistry) RegisterWorkflow(w engine.WorkflowDefinition) {
	r.workflows[w.ID] = w
}

func (r *InMemRegistry) RegisterGeneric(id string, data json.RawMessage) {
	r.generics[id] = data
}

func (r *InMemRegistry) GetTaskTemplate(id string) (orchestrator.TaskTemplate, bool) {
	t, ok := r.tasks[id]
	return t, ok
}

func (r *InMemRegistry) GetSubTaskTemplate(id string) (orchestrator.SubTaskTemplate, bool) {
	st, ok := r.subtasks[id]
	return st, ok
}

func (r *InMemRegistry) GetWorkflow(id string) (engine.WorkflowDefinition, bool) {
	w, ok := r.workflows[id]
	return w, ok
}

func (r *InMemRegistry) GetGenericTemplate(id string) (json.RawMessage, bool) {
	g, ok := r.generics[id]
	return g, ok
}
