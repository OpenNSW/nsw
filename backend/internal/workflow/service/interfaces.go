package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type TemplateProvider interface {
	// GetWorkflowTemplateMapByHSCodeIDAndFlow retrieves the workflow template map associated with a given HS code and consignment flow.
	GetWorkflowTemplateMapByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplateMap, error)

	// GetWorkflowTemplateByHSCodeIDAndFlow (Legacy) retrieves the traditional workflow template associated with a given HS code and consignment flow.
	GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error)

	// GetGoWorkflowTemplateByID retrieves a go-workflow template by its ID.
	GetGoWorkflowTemplateByID(ctx context.Context, id uuid.UUID) (*model.GoWorkflowTemplate, error)

	// GetWorkflowTemplateByID retrieves a workflow template by its ID.
	GetWorkflowTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowTemplate, error)

	// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
	GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]model.WorkflowNodeTemplate, error)

	// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
	GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error)

	// GetEndNodeTemplate retrieves the special end node template.
	GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error)
}

// Compile-time interface compliance checks
var _ TemplateProvider = (*TemplateService)(nil)
