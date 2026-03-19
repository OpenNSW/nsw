package manager

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type WorkflowNodeService struct {
	db *gorm.DB
}

// NewWorkflowNodeService creates a new instance of WorkflowNodeService.
func NewWorkflowNodeService(db *gorm.DB) *WorkflowNodeService {
	return &WorkflowNodeService{db: db}
}

// GetWorkflowNodeByID retrieves a workflow node by its ID.
func (s *WorkflowNodeService) GetWorkflowNodeByID(ctx context.Context, nodeID string) (*model.WorkflowNode, error) {
	var node model.WorkflowNode
	result := s.db.WithContext(ctx).Where("id = ?", nodeID).First(&node)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow node: %w", result.Error)
	}
	return &node, nil
}

// GetWorkflowNodeByIDInTx retrieves a workflow node by its ID within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID string) (*model.WorkflowNode, error) {
	var node model.WorkflowNode
	result := tx.WithContext(ctx).Where("id = ?", nodeID).First(&node)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow node in transaction: %w", result.Error)
	}
	return &node, nil
}

// GetWorkflowNodesByIDsInTx retrieves multiple workflow nodes by their IDs within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []string) ([]model.WorkflowNode, error) {
	var nodes []model.WorkflowNode
	result := tx.WithContext(ctx).Where("id IN ?", nodeIDs).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes in transaction: %w", result.Error)
	}
	return nodes, nil
}

// CreateWorkflowNodesInTx creates multiple workflow nodes within a transaction.
func (s *WorkflowNodeService) CreateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) ([]model.WorkflowNode, error) {
	if len(nodes) == 0 {
		return []model.WorkflowNode{}, nil
	}

	result := tx.WithContext(ctx).Create(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create workflow nodes in transaction: %w", result.Error)
	}

	return nodes, nil
}

// UpdateWorkflowNodesInTx updates multiple workflow nodes within a transaction.
func (s *WorkflowNodeService) UpdateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// Update each node individually to avoid duplicate inserts
	// First fetch the existing record, then update it to ensure GORM tracks it properly
	for _, node := range nodes {
		// Fetch the existing node from database
		var existingNode model.WorkflowNode
		result := tx.WithContext(ctx).Where("id = ?", node.ID).First(&existingNode)
		if result.Error != nil {
			return fmt.Errorf("failed to find workflow node %s for update: %w", node.ID, result.Error)
		}

		// Update the fields
		existingNode.State = node.State
		existingNode.ExtendedState = node.ExtendedState
		existingNode.Outcome = node.Outcome
		existingNode.DependsOn = node.DependsOn
		existingNode.UnlockConfiguration = node.UnlockConfiguration

		// Save the updated node
		result = tx.WithContext(ctx).Save(&existingNode)
		if result.Error != nil {
			return fmt.Errorf("failed to update workflow node %s in transaction: %w", node.ID, result.Error)
		}
	}
	return nil
}

// GetWorkflowNodesByWorkflowIDInTx retrieves all workflow nodes associated with a given workflow ID within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByWorkflowIDInTx(ctx context.Context, tx *gorm.DB, workflowID string) ([]model.WorkflowNode, error) {
	var nodes []model.WorkflowNode
	result := tx.WithContext(ctx).Where("workflow_id = ?", workflowID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for workflow %s in transaction: %w", workflowID, result.Error)
	}
	return nodes, nil
}

// GetWorkflowNodesByWorkflowIDsInTx retrieves all workflow nodes associated with multiple workflow IDs within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByWorkflowIDsInTx(ctx context.Context, tx *gorm.DB, workflowIDs []string) ([]model.WorkflowNode, error) {
	if len(workflowIDs) == 0 {
		return []model.WorkflowNode{}, nil
	}

	var nodes []model.WorkflowNode
	result := tx.WithContext(ctx).Where("workflow_id IN ?", workflowIDs).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for %d workflows in transaction: %w", len(workflowIDs), result.Error)
	}
	return nodes, nil
}

// CountIncompleteNodesByWorkflowID counts the number of incomplete workflow nodes for a given workflow.
func (s *WorkflowNodeService) CountIncompleteNodesByWorkflowID(ctx context.Context, tx *gorm.DB, workflowID string) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).
		Model(&model.WorkflowNode{}).
		Where("workflow_id = ? AND state != ?", workflowID, model.WorkflowNodeStateCompleted).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count incomplete nodes for workflow %s: %w", workflowID, err)
	}
	return count, nil
}
