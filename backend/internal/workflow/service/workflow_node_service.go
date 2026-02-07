package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type WorkflowNodeService struct {
	// db *gorm.DB
}

// NewWorkflowNodeService creates a new instance of WorkflowNodeService.
func NewWorkflowNodeService(db *gorm.DB) *WorkflowNodeService {
	return &WorkflowNodeService{}
}

// GetWorkflowNodeByIDInTx retrieves a workflow node by its ID within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*model.WorkflowNode, error) {
	var node model.WorkflowNode
	result := tx.WithContext(ctx).Where("id = ?", nodeID).First(&node)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow node in transaction: %w", result.Error)
	}
	return &node, nil
}

// GetWorkflowNodesByIDsInTx retrieves multiple workflow nodes by their IDs within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []uuid.UUID) ([]model.WorkflowNode, error) {
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
		existingNode.DependsOn = node.DependsOn
		if existingNode.DependsOn == nil {
			existingNode.DependsOn = model.UUIDArray{}
		}

		// Save the updated node
		result = tx.WithContext(ctx).Save(&existingNode)
		if result.Error != nil {
			return fmt.Errorf("failed to update workflow node %s in transaction: %w", node.ID, result.Error)
		}
	}
	return nil
}

// GetWorkflowNodesByConsignmentIDInTx retrieves all workflow nodes associated with a given consignment ID within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByConsignmentIDInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) ([]model.WorkflowNode, error) {
	var nodes []model.WorkflowNode
	result := tx.WithContext(ctx).Where("consignment_id = ?", consignmentID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes in transaction: %w", result.Error)
	}
	return nodes, nil
}

// GetWorkflowNodesByConsignmentIDsInTx retrieves all workflow nodes associated with multiple consignment IDs within a transaction.
// This method is optimized for batch operations to avoid N+1 query problems.
func (s *WorkflowNodeService) GetWorkflowNodesByConsignmentIDsInTx(ctx context.Context, tx *gorm.DB, consignmentIDs []uuid.UUID) ([]model.WorkflowNode, error) {
	if len(consignmentIDs) == 0 {
		return []model.WorkflowNode{}, nil
	}

	var nodes []model.WorkflowNode
	result := tx.WithContext(ctx).Where("consignment_id IN ?", consignmentIDs).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for %d consignments in transaction: %w", len(consignmentIDs), result.Error)
	}
	return nodes, nil
}

// CountIncompleteNodesByConsignmentID counts the number of incomplete workflow nodes for a given consignment.
// This is more efficient than fetching all nodes when only checking completion status.
func (s *WorkflowNodeService) CountIncompleteNodesByConsignmentID(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).
		Model(&model.WorkflowNode{}).
		Where("consignment_id = ? AND state != ?", consignmentID, model.WorkflowNodeStateCompleted).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count incomplete nodes for consignment %s: %w", consignmentID, err)
	}
	return count, nil
}
