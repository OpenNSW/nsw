package service

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// TemplateProvider defines the interface for retrieving workflow templates.
// This abstraction allows for easier testing and flexibility in template storage.
type TemplateProvider interface {
	// GetWorkflowTemplateByHSCodeIDAndFlow retrieves the workflow template associated with a given HS code and consignment flow.
	GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error)

	// GetWorkflowTemplateByHSCodeIDAndFlowV2 retrieves the workflow template associated with a given HS code and consignment flow.
	GetWorkflowTemplateByHSCodeIDAndFlowV2(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplateV2, error)

	// GetWorkflowTemplateByID retrieves a workflow template by its ID.
	GetWorkflowTemplateByID(ctx context.Context, id string) (*model.WorkflowTemplate, error)

	// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
	GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []string) ([]model.WorkflowNodeTemplate, error)

	// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
	GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error)

	// GetEndNodeTemplate retrieves the special end node template.
	GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error)
}

// Compile-time interface compliance checks
var _ TemplateProvider = (*TemplateService)(nil)
