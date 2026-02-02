package r_service

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConsignmentService struct {
	db                  *gorm.DB
	templateService     *TemplateService
	workflowNodeService *WorkflowNodeService
}

// NewConsignmentService creates a new instance of ConsignmentService.
func NewConsignmentService(db *gorm.DB) *ConsignmentService {
	return &ConsignmentService{
		db:                  db,
		templateService:     NewTemplateService(db),
		workflowNodeService: NewWorkflowNodeService(db),
	}
}

// InitializeConsignment initializes the consignment based on the provided creation request.
// Returns the (created consignment response DTO and the new READY workflow nodes) or an error if the operation fails.
func (s *ConsignmentService) InitializeConsignment(ctx context.Context, createReq *r_model.CreateConsignmentDTO) (*r_model.ConsignmentResponseDTO, []r_model.WorkflowNode, error) {
	if createReq == nil {
		return nil, nil, fmt.Errorf("create request cannot be nil")
	}
	if len(createReq.Items) == 0 {
		return nil, nil, fmt.Errorf("consignment must have at least one item")
	}
	if createReq.TraderID == nil {
		return nil, nil, fmt.Errorf("trader ID cannot be empty")
	}

	consignment, newReadyWorkflowNodes, err := s.initializeConsignmentInTx(ctx, createReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize consignment: %w", err)
	}

	return consignment, newReadyWorkflowNodes, nil
}

// initializeConsignmentInTx initializes the consignment within a transaction.
func (s *ConsignmentService) initializeConsignmentInTx(ctx context.Context, createReq *r_model.CreateConsignmentDTO) (*r_model.ConsignmentResponseDTO, []r_model.WorkflowNode, error) {
	consignment := &r_model.Consignment{
		Flow:          createReq.Flow,
		TraderID:      *createReq.TraderID,
		State:         r_model.ConsignmentStateInProgress,
		GlobalContext: createReq.GlobalContext,
	}

	var items []r_model.ConsignmentItem
	var workflowTemplates []r_model.WorkflowTemplate
	for _, itemDTO := range createReq.Items {
		item := r_model.ConsignmentItem{
			HSCodeID:     itemDTO.HSCodeID,
			ItemMetadata: itemDTO.ItemMetadata,
		}
		items = append(items, item)
		workflowTemplate, err := s.templateService.GetWorkflowTemplateByHSCodeIDAndFlow(ctx, itemDTO.HSCodeID, string(createReq.Flow))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template for HS code %s and flow %s: %w", itemDTO.HSCodeID, createReq.Flow, err)
		}
		workflowTemplates = append(workflowTemplates, *workflowTemplate)
	}
	consignment.Items = items

	// Initiate Transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create Consignment
	if err := tx.Create(consignment).Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to create consignment: %w", err)
	}

	// Create Workflow Nodes
	workflowNodes, newReadyWorkflowNodes, err := s.createWorkflowNodesInTx(ctx, tx, consignment.ID, workflowTemplates)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to create workflow nodes: %w", err)
	}

	// Commit Transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Prepare Response DTO
	responseDTO := &r_model.ConsignmentResponseDTO{
		ID:            consignment.ID,
		Flow:          consignment.Flow,
		TraderID:      consignment.TraderID,
		State:         consignment.State,
		Items:         consignment.Items,
		GlobalContext: consignment.GlobalContext,
		CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes: workflowNodes,
	}

	return responseDTO, newReadyWorkflowNodes, nil
}

// createWorkflowNodesInTx builds workflow nodes for the consignment within a transaction.
func (s *ConsignmentService) createWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID, workflowTemplates []r_model.WorkflowTemplate) ([]r_model.WorkflowNode, []r_model.WorkflowNode, error) {
	var workflowNodes []r_model.WorkflowNode
	var uniqueNodeTemplates = make(map[uuid.UUID]bool)
	for _, wt := range workflowTemplates {
		nodeTemplateIDs := wt.GetNodeTemplateIDs()
		for _, nodeTemplateID := range nodeTemplateIDs {
			if _, exists := uniqueNodeTemplates[nodeTemplateID]; !exists {
				uniqueNodeTemplates[nodeTemplateID] = true
				workflowNode := r_model.WorkflowNode{
					ConsignmentID:          consignmentID,
					WorkflowNodeTemplateID: nodeTemplateID,
					State:                  r_model.WorkflowNodeStateReady,
					DependsOn:              []uuid.UUID{},
				}
				workflowNodes = append(workflowNodes, workflowNode)
			}
		}
	}

	// Save Workflow Nodes to get their IDs
	createdNodes, err := s.workflowNodeService.CreateWorkflowNodesInTx(ctx, tx, workflowNodes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create workflow nodes in transaction: %w", err)
	}

	// Get All Workflow Node Templates to set dependencies
	var workflowNodeTemplates []r_model.WorkflowNodeTemplate
	workflowNodeTemplateIDs := make([]uuid.UUID, 0, len(uniqueNodeTemplates))
	for id := range uniqueNodeTemplates {
		workflowNodeTemplateIDs = append(workflowNodeTemplateIDs, id)
	}
	workflowNodeTemplates, err = s.templateService.GetWorkflowNodeTemplatesByIDs(ctx, workflowNodeTemplateIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve workflow node templates: %w", err)
	}
	templateMap := make(map[uuid.UUID]r_model.WorkflowNodeTemplate)
	for _, nt := range workflowNodeTemplates {
		templateMap[nt.ID] = nt
	}
	// Set Dependencies
	for i, node := range createdNodes {
		template, exists := templateMap[node.WorkflowNodeTemplateID]
		if !exists {
			return nil, nil, fmt.Errorf("workflow node template with ID %s not found", node.WorkflowNodeTemplateID)
		}
		var dependsOnNodeIDs []uuid.UUID
		for _, dependsOnTemplateID := range template.DependsOn {
			for _, n := range createdNodes {
				if n.WorkflowNodeTemplateID == dependsOnTemplateID {
					dependsOnNodeIDs = append(dependsOnNodeIDs, n.ID)
					break
				}
			}
		}
		createdNodes[i].DependsOn = dependsOnNodeIDs
	}

	// If there are nodes without dependencies, set them to READY
	var newReadyNodes []r_model.WorkflowNode
	for i, node := range createdNodes {
		if len(node.DependsOn) == 0 {
			createdNodes[i].State = r_model.WorkflowNodeStateReady
			newReadyNodes = append(newReadyNodes, createdNodes[i])
		}
	}

	// Update Workflow Nodes with dependencies and states
	if err := s.workflowNodeService.UpdateWorkflowNodesInTx(ctx, tx, createdNodes); err != nil {
		return nil, nil, fmt.Errorf("failed to update workflow nodes with dependencies: %w", err)
	}

	return createdNodes, newReadyNodes, nil
}

// GetConsignmentByID retrieves a consignment by its ID from the database.
func (s *ConsignmentService) GetConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*r_model.ConsignmentResponseDTO, error) {
	var consignment r_model.Consignment
	result := s.db.WithContext(ctx).Preload("Items").First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment: %w", result.Error)
	}

	// Retrieve associated workflow nodes
	workflowNodes, err := s.workflowNodeService.GetWorkflowNodesByConsignmentIDInTx(ctx, s.db, consignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for consignment: %w", err)
	}

	responseDTO := &r_model.ConsignmentResponseDTO{
		ID:            consignment.ID,
		Flow:          consignment.Flow,
		TraderID:      consignment.TraderID,
		State:         consignment.State,
		Items:         consignment.Items,
		GlobalContext: consignment.GlobalContext,
		CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes: workflowNodes,
	}

	return responseDTO, nil
}

// GetConsignmentsByTraderID retrieves consignments associated with a specific trader ID.
func (s *ConsignmentService) GetConsignmentsByTraderID(ctx context.Context, traderID string) ([]r_model.ConsignmentResponseDTO, error) {
	var consignments []r_model.Consignment
	result := s.db.WithContext(ctx).Preload("Items").Where("trader_id = ?", traderID).Find(&consignments)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignments for trader %s: %w", traderID, result.Error)
	}

	var consignmentDTOs []r_model.ConsignmentResponseDTO
	for _, consignment := range consignments {
		// Retrieve associated workflow nodes
		workflowNodes, err := s.workflowNodeService.GetWorkflowNodesByConsignmentIDInTx(ctx, s.db, consignment.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve workflow nodes for consignment %s: %w", consignment.ID, err)
		}

		dto := r_model.ConsignmentResponseDTO{
			ID:            consignment.ID,
			Flow:          consignment.Flow,
			TraderID:      consignment.TraderID,
			State:         consignment.State,
			Items:         consignment.Items,
			GlobalContext: consignment.GlobalContext,
			CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
			WorkflowNodes: workflowNodes,
		}
		consignmentDTOs = append(consignmentDTOs, dto)
	}

	return consignmentDTOs, nil
}

// UpdateConsignment updates an existing consignment in the database.
func (s *ConsignmentService) UpdateConsignment(ctx context.Context, updateReq *r_model.UpdateConsignmentDTO) (*r_model.ConsignmentResponseDTO, error) {
	if updateReq == nil {
		return nil, fmt.Errorf("update request cannot be nil")
	}

	var consignment r_model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", updateReq.ConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment: %w", result.Error)
	}

	// Apply updates
	if updateReq.State != nil {
		consignment.State = *updateReq.State
	}
	if updateReq.AppendToGlobalContext != nil {
		if consignment.GlobalContext == nil {
			consignment.GlobalContext = make(map[string]any)
		}
		maps.Copy(consignment.GlobalContext, updateReq.AppendToGlobalContext)
		// TODO: Implement the global context key selection such that no overwriting occurs.
	}

	// Save updates
	saveResult := s.db.WithContext(ctx).Save(&consignment)
	if saveResult.Error != nil {
		return nil, fmt.Errorf("failed to update consignment: %w", saveResult.Error)
	}

	// Retrieve associated workflow nodes
	workflowNodes, err := s.workflowNodeService.GetWorkflowNodesByConsignmentIDInTx(ctx, s.db, consignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for consignment: %w", err)
	}

	// Prepare response DTO
	responseDTO := &r_model.ConsignmentResponseDTO{
		ID:            consignment.ID,
		Flow:          consignment.Flow,
		TraderID:      consignment.TraderID,
		State:         consignment.State,
		Items:         consignment.Items,
		GlobalContext: consignment.GlobalContext,
		CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes: workflowNodes,
	}

	return responseDTO, nil
}

// isAllWorkflowNodesCompleted checks if all workflow nodes for a consignment are in COMPLETED state.
func (s *ConsignmentService) isAllWorkflowNodesCompleted(ctx context.Context, consignmentID uuid.UUID) (bool, error) {
	workflowNodes, err := s.workflowNodeService.GetWorkflowNodesByConsignmentIDInTx(ctx, s.db, consignmentID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve workflow nodes for consignment %s: %w", consignmentID, err)
	}

	for _, node := range workflowNodes {
		if node.State != r_model.WorkflowNodeStateCompleted {
			return false, nil
		}
	}

	return true, nil
}

// UpdateWorkflowNodeStateAndPropagateChanges updates the state of a workflow node and propagates changes to dependent nodes and consignment state.
func (s *ConsignmentService) UpdateWorkflowNodeStateAndPropagateChanges(ctx context.Context, updateReq *r_model.UpdateWorkflowNodeDTO) error {
	if updateReq == nil {
		return fmt.Errorf("update request cannot be nil")
	}

	// Start a transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update the workflow node state and propagate changes
	if err := s.updateWorkflowNodeStateAndPropagateChangesInTx(ctx, tx, updateReq); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update workflow node state and propagate changes: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// updateWorkflowNodeStateAndPropagateChangesInTx updates the workflow node state and propagates changes within a transaction.
func (s *ConsignmentService) updateWorkflowNodeStateAndPropagateChangesInTx(ctx context.Context, tx *gorm.DB, updateReq *r_model.UpdateWorkflowNodeDTO) error {
	// Get the workflow node
	workflowNode, err := s.workflowNodeService.GetWorkflowNodeByIDInTx(ctx, tx, updateReq.WorkflowNodeID)
	if err != nil {
		return fmt.Errorf("failed to retrieve workflow node: %w", err)
	}

	if workflowNode.State != updateReq.State && updateReq.State == r_model.WorkflowNodeStateCompleted {
		var nodesToUpdate []r_model.WorkflowNode

		// Update the current node to COMPLETED
		workflowNode.State = r_model.WorkflowNodeStateCompleted
		nodesToUpdate = append(nodesToUpdate, *workflowNode)

		// Retrieve all workflow nodes for the consignment
		allNodes, err := s.workflowNodeService.GetWorkflowNodesByConsignmentIDInTx(ctx, tx, workflowNode.ConsignmentID)
		if err != nil {
			return fmt.Errorf("failed to retrieve workflow nodes for consignment: %w", err)

		}

		// Update dependent nodes if their dependencies are met
		for _, node := range allNodes {
			if node.State == r_model.WorkflowNodeStateLocked {
				dependenciesMet := true
				for _, dependsOnID := range node.DependsOn {
					depNode, err := s.workflowNodeService.GetWorkflowNodeByIDInTx(ctx, tx, dependsOnID)
					if err != nil {
						return fmt.Errorf("failed to retrieve dependent workflow node: %w", err)
					}
					if depNode.State != r_model.WorkflowNodeStateCompleted {
						dependenciesMet = false
						break
					}
				}
				if dependenciesMet {
					node.State = r_model.WorkflowNodeStateReady
					nodesToUpdate = append(nodesToUpdate, node)
				}
			}
		}

		// Update all affected nodes in the database
		if err := s.workflowNodeService.UpdateWorkflowNodesInTx(ctx, tx, nodesToUpdate); err != nil {
			return fmt.Errorf("failed to update workflow nodes: %w", err)
		}

		// Check if all workflow nodes are completed to potentially update consignment state
		allCompleted, err := s.isAllWorkflowNodesCompleted(ctx, workflowNode.ConsignmentID)
		if err != nil {
			return fmt.Errorf("failed to check if all workflow nodes are completed: %w", err)
		}
		if allCompleted {
			var consignment r_model.Consignment
			result := tx.WithContext(ctx).First(&consignment, "id = ?", workflowNode.ConsignmentID)
			if result.Error != nil {
				return fmt.Errorf("failed to retrieve consignment: %w", result.Error)
			}

			consignment.State = r_model.ConsignmentStateFinished
			saveResult := tx.WithContext(ctx).Save(&consignment)
			if saveResult.Error != nil {
				return fmt.Errorf("failed to update consignment state: %w", saveResult.Error)
			}
		}
	}

	// If there's global context to append, update the consignment
	if len(updateReq.AppendGlobalContext) > 0 {
		var consignment r_model.Consignment
		result := tx.WithContext(ctx).First(&consignment, "id = ?", workflowNode.ConsignmentID)
		if result.Error != nil {
			return fmt.Errorf("failed to retrieve consignment: %w", result.Error)
		}

		if consignment.GlobalContext == nil {
			consignment.GlobalContext = make(map[string]any)
		}
		maps.Copy(consignment.GlobalContext, updateReq.AppendGlobalContext)

		// Save the updated consignment
		saveResult := tx.WithContext(ctx).Save(&consignment)
		if saveResult.Error != nil {
			return fmt.Errorf("failed to update consignment global context: %w", saveResult.Error)
		}
	}

	return nil
}
