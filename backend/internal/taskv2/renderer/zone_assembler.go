package renderer

import (
	"context"
	"encoding/json"
	"fmt"

	tfrenderer "github.com/OpenNSW/nsw-task-flow/renderer"
	"github.com/OpenNSW/nsw-task-flow/store"
)

// ZoneViewAssembler builds the ZoneView payload served by GET /api/v1/tasks/{id}.
// It delegates the per-zone projection to TaskRenderer and pulls state→actions
// from the same render config blob.
type ZoneViewAssembler struct {
	inner *TaskRenderer
}

func NewZoneViewAssembler(inner *TaskRenderer) *ZoneViewAssembler {
	return &ZoneViewAssembler{inner: inner}
}

func (a *ZoneViewAssembler) Assemble(ctx context.Context, record store.TaskRecord) (ZoneView, error) {
	view, err := a.inner.Render(ctx, record.RenderConfig, tfrenderer.Facts{
		State: record.State,
		Data:  record.Data,
	})
	if err != nil {
		return ZoneView{}, fmt.Errorf("zone assembler: render: %w", err)
	}

	var cfg TaskTemplateConfig
	if len(record.RenderConfig) > 0 {
		if err := json.Unmarshal(record.RenderConfig, &cfg); err != nil {
			return ZoneView{}, fmt.Errorf("zone assembler: decode states: %w", err)
		}
	}

	return ZoneView{
		TaskID:    record.TaskID,
		TaskType:  record.TaskType,
		State:     record.State,
		Actions:   cfg.States[record.State].Actions,
		View:      view,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}
