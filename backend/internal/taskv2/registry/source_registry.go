package registry

import (
	"context"
	"encoding/json"
	"log/slog"

	engine "github.com/OpenNSW/go-temporal-workflow"
	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw/pkg/templatesource"
)

// SourceRegistry adapts a templatesource.Source to the orchestrator's
// TaskTemplateRegistry contract. Blobs are keyed by their embedded "id"
// field; this registry routes lookups by attempting to unmarshal into the
// shape each Get* method expects and returning ok=false on a shape mismatch.
//
// The TaskTemplate wrapper around a child workflow is synthesized on the fly:
// when GetTaskTemplate sees a blob whose shape is a workflow definition, it
// returns TaskTemplate{ID: id, WorkflowID: id}. The same id then resolves the
// real workflow on the follow-up GetWorkflow call.
type SourceRegistry struct {
	src templatesource.Source
	ctx context.Context
}

func NewSourceRegistry(ctx context.Context, src templatesource.Source) *SourceRegistry {
	return &SourceRegistry{src: src, ctx: ctx}
}

func (r *SourceRegistry) fetch(id string) (json.RawMessage, bool) {
	data, ok, err := r.src.GetTemplate(r.ctx, id)
	if err != nil {
		slog.Warn("template fetch failed", "id", id, "err", err)
		return nil, false
	}
	return data, ok
}

func (r *SourceRegistry) GetTaskTemplate(id string) (orchestrator.TaskTemplate, bool) {
	data, ok := r.fetch(id)
	if !ok {
		return orchestrator.TaskTemplate{}, false
	}
	if !looksLikeWorkflow(data) {
		return orchestrator.TaskTemplate{}, false
	}
	return orchestrator.TaskTemplate{
		ID:         id,
		WorkflowID: id,
	}, true
}

func (r *SourceRegistry) GetSubTaskTemplate(id string) (orchestrator.SubTaskTemplate, bool) {
	data, ok := r.fetch(id)
	if !ok {
		return orchestrator.SubTaskTemplate{}, false
	}
	var st orchestrator.SubTaskTemplate
	if err := json.Unmarshal(data, &st); err != nil {
		return orchestrator.SubTaskTemplate{}, false
	}
	if st.TaskType == "" {
		return orchestrator.SubTaskTemplate{}, false
	}
	return st, true
}

func (r *SourceRegistry) GetWorkflow(id string) (engine.WorkflowDefinition, bool) {
	data, ok := r.fetch(id)
	if !ok {
		return engine.WorkflowDefinition{}, false
	}
	var wf engine.WorkflowDefinition
	if err := json.Unmarshal(data, &wf); err != nil {
		return engine.WorkflowDefinition{}, false
	}
	if len(wf.Nodes) == 0 {
		return engine.WorkflowDefinition{}, false
	}
	return wf, true
}

func (r *SourceRegistry) GetGenericTemplate(id string) (json.RawMessage, bool) {
	return r.fetch(id)
}

// looksLikeWorkflow peeks at the blob's top-level shape without a full
// unmarshal. A workflow definition always carries a non-empty "nodes" array.
func looksLikeWorkflow(data json.RawMessage) bool {
	var probe struct {
		Nodes []json.RawMessage `json:"nodes"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	return len(probe.Nodes) > 0
}

var _ orchestrator.TaskTemplateRegistry = (*SourceRegistry)(nil)
