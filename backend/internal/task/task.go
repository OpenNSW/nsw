package task

import (
	"context"
	"encoding/json"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
)

// InitPayload represents the data required to initialize a task in the ExecutionUnit Manager system.
type InitPayload struct {
	TaskID     uuid.UUID        `json:"taskId" binding:"required"` // Unique identifier of the task
	Type       Type             `json:"type" binding:"required"`   // Type of the task
	Status     model.TaskStatus `json:"status" binding:"required"` // Current status of the task
	CommandSet json.RawMessage  `json:"config" binding:"required"` // Configuration specific to the task
}

// ActiveTask represents a task that is currently active in the system.
type ActiveTask struct {
	TaskID          uuid.UUID
	TaskExecutionID uuid.UUID
	Type            Type
	Status          model.TaskStatus
	Executor        ExecutionUnit
}

func NewActiveTask(payload InitPayload, executor ExecutionUnit) *ActiveTask {
	return &ActiveTask{
		TaskID:          payload.TaskID,
		TaskExecutionID: uuid.New(),
		Type:            payload.Type,
		Status:          payload.Status,
		Executor:        executor,
	}
}

func (a *ActiveTask) GetID() uuid.UUID {
	return a.TaskID
}

func (a *ActiveTask) GetType() Type {
	return a.Type
}

func (a *ActiveTask) GetStatus() model.TaskStatus {
	return a.Status
}

func (a *ActiveTask) SetStatus(status model.TaskStatus) {
	a.Status = status
}

func (a *ActiveTask) IsExecutable() bool {
	return a.Status == model.TaskStatusReady
}

func (a *ActiveTask) Execute(ctx context.Context, payload interface{}) (*ExecutionResult, error) {
	return a.Executor.Execute(ctx, payload)
}
