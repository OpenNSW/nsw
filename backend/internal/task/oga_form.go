package task

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type OGAFormTask struct {
	CommandSet interface{}
}

func (t *OGAFormTask) Execute(_ context.Context, payload *ExecutionPayload) (*ExecutionResult, error) {
	// If payload is provided, it's a callback from OGA service (or internal tool)
	if payload != nil && payload.Action == "OGA_VERIFICATION" {
		// Process OGA decision
		contentMap, ok := payload.Content.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid content format")
		}

		decision, _ := contentMap["decision"].(string)
		
		status := model.TaskStatusCompleted
		if decision == "REJECTED" {
			status = model.TaskStatusRejected
		}

		return &ExecutionResult{
			Status:  status,
			Message: "OGA verification received",
			Data:    contentMap,
		}, nil
	}

	// Default initialization logic (when task is first activated)
	// 1. Route to external OGA system (AYUSCUDA, etc.)
	// TODO: Implement actual HTTP call to external OGA API
	// For now, we'll simulate the routing

	// 2. Update task status to IN_PROGRESS (will be updated to COMPLETED/REJECTED when OGA notifies)
	// The actual status update happens when OGA calls NotifyTaskCompletion

	// 3. Return IN_PROGRESS status
	// Executor Manager will notify Workflow Manager with INPROGRESS state
	return &ExecutionResult{
		Status:  model.TaskStatusInProgress, // Submitted to external system, waiting for OGA response
		Message: "OGA form routed to external system",
	}, nil
}
