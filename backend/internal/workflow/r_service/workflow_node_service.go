package r_service

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkflowNodeService struct {
	// db *gorm.DB
}

// NewWorkflowNodeService creates a new instance of WorkflowNodeService.
func NewWorkflowNodeService(db *gorm.DB) *WorkflowNodeService {
	return &WorkflowNodeService{
		// db: db,
	}
}

// // GetWorkflowNodeByID retrieves a workflow node by its ID.
// func (s *WorkflowNodeService) GetWorkflowNodeByID(ctx context.Context, nodeID uuid.UUID) (*r_model.WorkflowNode, error) {
// 	var node r_model.WorkflowNode
// 	result := s.db.WithContext(ctx).Where("id = ?", nodeID).First(&node)
// 	if result.Error != nil {
// 		return nil, fmt.Errorf("failed to retrieve workflow node: %w", result.Error)
// 	}
// 	return &node, nil
// }

// // CreateWorkflowNodes creates multiple workflow nodes in the database.
// func (s *WorkflowNodeService) CreateWorkflowNodes(ctx context.Context, nodes []r_model.WorkflowNode) ([]r_model.WorkflowNode, error) {
// 	if len(nodes) == 0 {
// 		return []r_model.WorkflowNode{}, nil
// 	}

// 	result := s.db.WithContext(ctx).Create(&nodes)
// 	if result.Error != nil {
// 		return nil, fmt.Errorf("failed to create workflow nodes: %w", result.Error)
// 	}

// 	return nodes, nil
// }

// // GetWorkflowNodesByConsignmentID retrieves all workflow nodes associated with a given consignment ID.
// func (s *WorkflowNodeService) GetWorkflowNodesByConsignmentID(ctx context.Context, consignmentID string) ([]r_model.WorkflowNode, error) {
// 	var nodes []r_model.WorkflowNode
// 	result := s.db.WithContext(ctx).Where("consignment_id = ?", consignmentID).Find(&nodes)
// 	if result.Error != nil {
// 		return nil, fmt.Errorf("failed to retrieve workflow nodes: %w", result.Error)
// 	}
// 	return nodes, nil
// }

// // UpdateWorkflowNodes updates multiple workflow nodes in the database.
// func (s *WorkflowNodeService) UpdateWorkflowNodes(ctx context.Context, nodes []r_model.WorkflowNode) error {
// 	if len(nodes) == 0 {
// 		return nil
// 	}

// 	result := s.db.WithContext(ctx).Save(&nodes)
// 	if result.Error != nil {
// 		return fmt.Errorf("failed to update workflow nodes: %w", result.Error)
// 	}

// 	return nil
// }

// GetWorkflowNodeByIDInTx retrieves a workflow node by its ID within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*r_model.WorkflowNode, error) {
	var node r_model.WorkflowNode
	result := tx.WithContext(ctx).Where("id = ?", nodeID).First(&node)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow node in transaction: %w", result.Error)
	}
	return &node, nil
}

// GetWorkflowNodesByIDsInTx retrieves multiple workflow nodes by their IDs within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []uuid.UUID) ([]r_model.WorkflowNode, error) {
	var nodes []r_model.WorkflowNode
	result := tx.WithContext(ctx).Where("id IN ?", nodeIDs).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes in transaction: %w", result.Error)
	}
	return nodes, nil
}

// CreateWorkflowNodesInTx creates multiple workflow nodes within a transaction.
func (s *WorkflowNodeService) CreateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []r_model.WorkflowNode) ([]r_model.WorkflowNode, error) {
	if len(nodes) == 0 {
		return []r_model.WorkflowNode{}, nil
	}

	result := tx.WithContext(ctx).Create(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create workflow nodes in transaction: %w", result.Error)
	}

	return nodes, nil
}

// UpdateWorkflowNodesInTx updates multiple workflow nodes within a transaction.
func (s *WorkflowNodeService) UpdateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []r_model.WorkflowNode) error {
	if len(nodes) == 0 {
		return nil
	}

	result := tx.WithContext(ctx).Save(&nodes)
	if result.Error != nil {
		return fmt.Errorf("failed to update workflow nodes in transaction: %w", result.Error)
	}
	return nil
}

// GetWorkflowNodesByConsignmentIDInTx retrieves all workflow nodes associated with a given consignment ID within a transaction.
func (s *WorkflowNodeService) GetWorkflowNodesByConsignmentIDInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) ([]r_model.WorkflowNode, error) {
	var nodes []r_model.WorkflowNode
	result := tx.WithContext(ctx).Where("consignment_id = ?", consignmentID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes in transaction: %w", result.Error)
	}
	return nodes, nil
}
