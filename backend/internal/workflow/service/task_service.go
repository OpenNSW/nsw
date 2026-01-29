package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskService struct {
	db *gorm.DB
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{db: db}
}

// CreateNodes creates multiple nodes in the database.
func (s *TaskService) CreateNodes(ctx context.Context, nodes []model.Node) ([]uuid.UUID, error) {
	if len(nodes) == 0 {
		return []uuid.UUID{}, nil
	}

	result := s.db.WithContext(ctx).Create(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create nodes: %w", result.Error)
	}

	nodeIDs := make([]uuid.UUID, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
	}

	return nodeIDs, nil
}

// GetNodesByConsignmentID retrieves all nodes associated with a given consignment ID.
func (s *TaskService) GetNodesByConsignmentID(ctx context.Context, consignmentID uuid.UUID) ([]model.Node, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}

	var nodes []model.Node
	result := s.db.WithContext(ctx).Where("consignment_id = ?", consignmentID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve nodes: %w", result.Error)
	}
	// Return empty slice instead of error when no nodes found
	return nodes, nil
}

// GetNodeByID retrieves a node by its ID.
func (s *TaskService) GetNodeByID(ctx context.Context, nodeID uuid.UUID) (*model.Node, error) {
	if nodeID == uuid.Nil {
		return nil, fmt.Errorf("node ID cannot be nil")
	}

	var node model.Node
	result := s.db.WithContext(ctx).First(&node, "id = ?", nodeID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("node %s not found", nodeID)
		}
		return nil, fmt.Errorf("failed to retrieve node: %w", result.Error)
	}
	return &node, nil
}

// UpdateNodes updates multiple nodes in the database within a transaction.
func (s *TaskService) UpdateNodes(ctx context.Context, nodes []model.Node) error {
	if len(nodes) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range nodes {
			if err := tx.Save(&nodes[i]).Error; err != nil {
				return fmt.Errorf("failed to update node %s: %w", nodes[i].ID, err)
			}
		}
		return nil
	})
}

// UpdateNodesInTx updates multiple nodes within an existing transaction.
// This is used when we're already in a transaction context.
func (s *TaskService) UpdateNodesInTx(ctx context.Context, tx *gorm.DB, nodes []*model.Node) error {
	if len(nodes) == 0 {
		return nil
	}

	for _, node := range nodes {
		if err := tx.WithContext(ctx).Save(node).Error; err != nil {
			return fmt.Errorf("failed to update node %s: %w", node.ID, err)
		}
	}
	return nil
}

// UpdateNodeStatus updates a single node's status.
func (s *TaskService) UpdateNodeStatus(ctx context.Context, nodeID uuid.UUID, status model.TaskStatus) error {
	if nodeID == uuid.Nil {
		return fmt.Errorf("node ID cannot be nil")
	}
	if status == "" {
		return fmt.Errorf("node status cannot be empty")
	}

	result := s.db.WithContext(ctx).Model(&model.Node{}).
		Where("id = ?", nodeID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update node status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("node %s not found", nodeID)
	}

	return nil
}

// CreateNodesInTx creates multiple nodes within an existing transaction.
func (s *TaskService) CreateNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.Node) ([]uuid.UUID, error) {
	if len(nodes) == 0 {
		return []uuid.UUID{}, nil
	}

	result := tx.WithContext(ctx).Create(&nodes)
	if result.Error != nil {
		return nil, result.Error
	}

	nodeIDs := make([]uuid.UUID, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
	}

	return nodeIDs, nil
}

// GetNodeByIDInTx retrieves a node by its ID within an existing transaction.
func (s *TaskService) GetNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*model.Node, error) {
	if nodeID == uuid.Nil {
		return nil, fmt.Errorf("node ID cannot be nil")
	}

	var node model.Node
	result := tx.WithContext(ctx).First(&node, "id = ?", nodeID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("node %s not found", nodeID)
		}
		return nil, fmt.Errorf("failed to retrieve node: %w", result.Error)
	}
	return &node, nil
}

// UpdateNodeInTx updates a single node within an existing transaction.
func (s *TaskService) UpdateNodeInTx(ctx context.Context, tx *gorm.DB, node *model.Node) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if err := tx.WithContext(ctx).Save(node).Error; err != nil {
		return fmt.Errorf("failed to update node %s: %w", node.ID, err)
	}
	return nil
}

// GetNodesByConsignmentIDInTx retrieves all nodes associated with a given consignment ID within an existing transaction.
func (s *TaskService) GetNodesByConsignmentIDInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) ([]model.Node, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}

	var nodes []model.Node
	result := tx.WithContext(ctx).Where("consignment_id = ?", consignmentID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve nodes: %w", result.Error)
	}
	return nodes, nil
}

// GetNodesByConsignmentIDAndDependencyStepID retrieves nodes by consignment ID that depend on a specific step ID.
// This uses the PostgreSQL JSONB ? operator (escaped as ?? in GORM) to check if a key exists in the depends_on JSONB column.
func (s *TaskService) GetNodesByConsignmentIDAndDependencyStepID(ctx context.Context, consignmentID uuid.UUID, stepID string) ([]model.Node, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}
	if stepID == "" {
		return nil, fmt.Errorf("step ID cannot be empty")
	}

	var nodes []model.Node
	result := s.db.WithContext(ctx).Where("consignment_id = ? AND depends_on ?? ?", consignmentID, stepID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve nodes: %w", result.Error)
	}
	return nodes, nil
}

// GetNodesByConsignmentIDAndDependencyStepIDInTx retrieves nodes by consignment ID that depend on a specific step ID within an existing transaction.
// This uses the PostgreSQL JSONB ? operator (escaped as ?? in GORM) to check if a key exists in the depends_on JSONB column.
func (s *TaskService) GetNodesByConsignmentIDAndDependencyStepIDInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID, stepID string) ([]model.Node, error) {
	if consignmentID == uuid.Nil {
		return nil, fmt.Errorf("consignment ID cannot be nil")
	}
	if stepID == "" {
		return nil, fmt.Errorf("step ID cannot be empty")
	}

	var nodes []model.Node
	result := tx.WithContext(ctx).Where("consignment_id = ? AND depends_on ?? ?", consignmentID, stepID).Find(&nodes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve nodes: %w", result.Error)
	}
	return nodes, nil
}
