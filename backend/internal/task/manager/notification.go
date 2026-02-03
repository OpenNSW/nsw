package manager

import (
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/google/uuid"
)

type WorkflowManagerNotification struct {
	TaskID              uuid.UUID
	UpdatedState        *plugin.State
	AppendGlobalContext map[string]any
}
