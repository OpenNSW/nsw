package renderer

import (
	"time"

	"github.com/OpenNSW/nsw-task-flow/renderer"
)

// Action mirrors the trader-app Action discriminated union. Kind selects which
// of Command (submit_form) or Action (task_action) carries the payload.
type Action struct {
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	Command string `json:"command,omitempty"`
	Action  string `json:"action,omitempty"`
	Variant string `json:"variant,omitempty"`
}

// StateView declares what affordances a task offers while it sits in a given
// state. Empty (or missing) means the state is terminal / non-interactive.
type StateView struct {
	Actions []Action `json:"actions,omitempty"`
}

// TaskTemplateConfig is the taskv2-level view of a render.json blob. The
// blueprint fields (sections, etc.) are decoded by the inner TaskRenderer and
// not modeled here — only States is consumed by the assembler.
type TaskTemplateConfig struct {
	States map[string]StateView `json:"states,omitempty"`
}

// ZoneView is the wire shape the trader-app's zone renderer consumes.
type ZoneView struct {
	TaskID    string                `json:"task_id"`
	TaskType  string                `json:"task_type"`
	State     string                `json:"state"`
	Actions   []Action              `json:"actions,omitempty"`
	View      renderer.RenderResult `json:"view"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}
