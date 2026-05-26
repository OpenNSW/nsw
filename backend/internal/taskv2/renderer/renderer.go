package renderer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw-task-flow/renderer"
	"github.com/OpenNSW/nsw/pkg/uiprojector"
)

// TaskRenderer adapts uiprojector.Assembler to the nsw-task-flow renderer
// contract. The render config blob is interpreted as a uiprojector.Blueprint;
// the resulting Sections are translated to UIComponents. Section.ID and
// Section.Title are dropped — UIComponent has no slots for them.
type TaskRenderer struct {
	assembler *uiprojector.Assembler
}

func NewTaskRenderer(assembler *uiprojector.Assembler) *TaskRenderer {
	return &TaskRenderer{assembler: assembler}
}

func (r *TaskRenderer) Render(ctx context.Context, configRaw json.RawMessage, facts renderer.Facts) (renderer.RenderResult, error) {
	if len(configRaw) == 0 {
		return renderer.RenderResult{}, nil
	}

	var bp uiprojector.Blueprint
	if err := json.Unmarshal(configRaw, &bp); err != nil {
		return nil, fmt.Errorf("renderer: unmarshal blueprint: %w", err)
	}

	sections, err := r.assembler.Assemble(ctx, &bp, uiprojector.Facts{
		State: facts.State,
		Data:  facts.Data,
	})
	if err != nil {
		return nil, fmt.Errorf("renderer: assemble: %w", err)
	}

	result := make(renderer.RenderResult, len(sections))
	for slot, sec := range sections {
		var content any = sec.Content
		secType := string(sec.Type)
		if sec.Type == "PAYMENT" {
			secType = "MARKDOWN"
		}
		if secType == "MARKDOWN" {
			if str, ok := sec.Content.(string); ok {
				content = map[string]any{"content": str}
			}
		}
		payload, err := json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("renderer: marshal section %q: %w", slot, err)
		}
		result[slot] = renderer.UIComponent{
			Type:    secType,
			Payload: payload,
		}
	}
	return result, nil
}

var _ renderer.Renderer = (*TaskRenderer)(nil)
