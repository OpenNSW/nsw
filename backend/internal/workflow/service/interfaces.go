package service

import (
	"context"

	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
)

// TemplateProvider defines the interface for retrieving workflow templates.
// This abstraction allows for easier testing and flexibility in template storage.
type TemplateProvider interface {

	// GetWorkflowTemplateByIDV2 retrieves a workflow template by its ID.
	GetWorkflowTemplateByIDV2(ctx context.Context, id string) (*model.WorkflowTemplateV2, error)

	// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
	GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []string) ([]model.WorkflowNodeTemplate, error)

	// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
	GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error)
}
