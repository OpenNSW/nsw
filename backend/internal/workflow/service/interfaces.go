package service

import (
	"context"

	engine "github.com/OpenNSW/core/workflow"
	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
)

// WorkflowDefinitionProvider defines the interface for retrieving in-memory workflow definitions.
type WorkflowDefinitionProvider interface {
	GetWorkflow(id string) (engine.WorkflowDefinition, bool)
}

// TemplateProvider defines the interface for retrieving workflow templates.
// TODO: Clean this up. With all workflows moved to the file-backed template registry,
// we no longer need the database-backed TemplateService or this interface.
type TemplateProvider interface {
	// GetWorkflowTemplateByIDV2 retrieves a workflow template by its ID.
	GetWorkflowTemplateByIDV2(ctx context.Context, id string) (*model.WorkflowTemplateV2, error)

	// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
	GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []string) ([]model.WorkflowNodeTemplate, error)

	// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
	GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error)
}
