package manager

import (
	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/task/plugin"
)

type WorkflowManagerNotification struct {
	TaskID              uuid.UUID
	UpdatedState        *plugin.State
	AppendGlobalContext map[string]any
	ExtendedState       *string
	Outcome             *string // Outcome sub-state for COMPLETED transitions (e.g., APPROVED, REJECTED)
}
