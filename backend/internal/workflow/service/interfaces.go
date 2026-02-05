package service

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// TemplateProvider defines the interface for retrieving workflow templates.
// This abstraction allows for easier testing and flexibility in template storage.
type TemplateProvider interface {
	// GetWorkflowTemplateByHSCodeIDAndFlow retrieves the workflow template associated with a given HS code and consignment flow.
	GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error)

	// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
	GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]model.WorkflowNodeTemplate, error)

	// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
	GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error)
}

// WorkflowNodeRepository defines the interface for workflow node data access operations.
// This abstraction decouples business logic from database implementation details.
type WorkflowNodeRepository interface {
	// GetWorkflowNodeByIDInTx retrieves a workflow node by its ID within a transaction.
	GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*model.WorkflowNode, error)

	// GetWorkflowNodesByIDsInTx retrieves multiple workflow nodes by their IDs within a transaction.
	GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []uuid.UUID) ([]model.WorkflowNode, error)

	// CreateWorkflowNodesInTx creates multiple workflow nodes within a transaction.
	CreateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) ([]model.WorkflowNode, error)

	// UpdateWorkflowNodesInTx updates multiple workflow nodes within a transaction.
	UpdateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) error

	// GetWorkflowNodesByConsignmentIDInTx retrieves all workflow nodes associated with a given consignment ID within a transaction.
	GetWorkflowNodesByConsignmentIDInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) ([]model.WorkflowNode, error)

	// GetWorkflowNodesByConsignmentIDsInTx retrieves all workflow nodes associated with multiple consignment IDs within a transaction.
	GetWorkflowNodesByConsignmentIDsInTx(ctx context.Context, tx *gorm.DB, consignmentIDs []uuid.UUID) ([]model.WorkflowNode, error)

	// CountIncompleteNodesByConsignmentID counts the number of incomplete workflow nodes for a given consignment.
	CountIncompleteNodesByConsignmentID(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) (int64, error)
}

// Compile-time interface compliance checks
var _ TemplateProvider = (*TemplateService)(nil)
var _ WorkflowNodeRepository = (*WorkflowNodeService)(nil)
