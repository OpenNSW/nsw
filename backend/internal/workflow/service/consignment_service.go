package service

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// ConsignmentService handles consignment-related operations.
// It coordinates between workflow templates, nodes, and the state machine.
type ConsignmentService struct {
	db                          *gorm.DB
	templateProvider            TemplateProvider
	nodeRepo                    WorkflowNodeRepository
	stateMachine                *WorkflowNodeStateMachine
	preCommitValidationCallback func([]model.WorkflowNode, map[string]any) error
}

// SetPreCommitValidationCallback sets a callback to be executed before transaction commit
// This allows external validation (like task manager registration) to participate in the transaction
func (s *ConsignmentService) SetPreCommitValidationCallback(callback func([]model.WorkflowNode, map[string]any) error) {
	s.preCommitValidationCallback = callback
}

// NewConsignmentService creates a new instance of ConsignmentService with interface dependencies.
// This constructor allows for dependency injection and easier testing.
func NewConsignmentService(db *gorm.DB, templateProvider TemplateProvider, nodeRepo WorkflowNodeRepository) *ConsignmentService {
	return &ConsignmentService{
		db:               db,
		templateProvider: templateProvider,
		nodeRepo:         nodeRepo,
		stateMachine:     NewWorkflowNodeStateMachine(nodeRepo),
	}
}

// NewConsignmentServiceWithDefaults creates a new instance of ConsignmentService with concrete implementations.
// This is a convenience constructor for production use.
func NewConsignmentServiceWithDefaults(db *gorm.DB, templateService *TemplateService, workflowNodeService *WorkflowNodeService) *ConsignmentService {
	return NewConsignmentService(db, templateService, workflowNodeService)
}

// InitializeConsignment initializes the consignment based on the provided creation request.
// Returns the (created consignment response DTO and the new READY workflow nodes) or an error if the operation fails.
func (s *ConsignmentService) InitializeConsignment(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.ConsignmentResponseDTO, []model.WorkflowNode, error) {
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
func (s *ConsignmentService) initializeConsignmentInTx(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.ConsignmentResponseDTO, []model.WorkflowNode, error) {
	consignment := &model.Consignment{
		Flow:          createReq.Flow,
		TraderID:      *createReq.TraderID,
		State:         model.ConsignmentStateInProgress,
		GlobalContext: createReq.GlobalContext,
	}

	var items []model.ConsignmentItem
	var workflowTemplates []model.WorkflowTemplate
	for _, itemDTO := range createReq.Items {
		item := model.ConsignmentItem(itemDTO)
		items = append(items, item)
		workflowTemplate, err := s.templateProvider.GetWorkflowTemplateByHSCodeIDAndFlow(ctx, itemDTO.HSCodeID, createReq.Flow)
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

	// Execute pre-commit validation callback if set (e.g., task manager registration)
	// This ensures external dependencies are validated before committing the transaction
	if s.preCommitValidationCallback != nil && len(newReadyWorkflowNodes) > 0 {
		if err := s.preCommitValidationCallback(newReadyWorkflowNodes, consignment.GlobalContext); err != nil {
			tx.Rollback()
			return nil, nil, fmt.Errorf("pre-commit validation failed: %w", err)
		}
	}

	// Commit Transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Prepare Response DTO using the helper function
	responseDTO, err := s.buildConsignmentResponseDTO(ctx, consignment, workflowNodes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, newReadyWorkflowNodes, nil
}

// createWorkflowNodesInTx builds workflow nodes for the consignment within a transaction.
func (s *ConsignmentService) createWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID, workflowTemplates []model.WorkflowTemplate) ([]model.WorkflowNode, []model.WorkflowNode, error) {
	// Collect unique node template IDs from all workflow templates
	uniqueNodeTemplateIDs := make(map[uuid.UUID]bool)
	for _, wt := range workflowTemplates {
		nodeTemplateIDs := wt.GetNodeTemplateIDs()
		for _, nodeTemplateID := range nodeTemplateIDs {
			uniqueNodeTemplateIDs[nodeTemplateID] = true
		}
	}

	// Fetch all node templates in a single query
	nodeTemplateIDsList := make([]uuid.UUID, 0, len(uniqueNodeTemplateIDs))
	for id := range uniqueNodeTemplateIDs {
		nodeTemplateIDsList = append(nodeTemplateIDsList, id)
	}

	nodeTemplates, err := s.templateProvider.GetWorkflowNodeTemplatesByIDs(ctx, nodeTemplateIDsList)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve workflow node templates: %w", err)
	}

	// Delegate to the state machine for node initialization
	return s.stateMachine.InitializeNodesFromTemplates(ctx, tx, consignmentID, nodeTemplates)
}

// GetConsignmentByID retrieves a consignment by its ID from the database.
func (s *ConsignmentService) GetConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*model.ConsignmentResponseDTO, error) {
	var consignment model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s: %w", consignmentID, result.Error)
	}

	// Retrieve associated workflow nodes
	workflowNodes, err := s.nodeRepo.GetWorkflowNodesByConsignmentIDInTx(ctx, s.db, consignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for consignment %s: %w", consignmentID, err)
	}

	// Build response DTO using the helper function
	responseDTO, err := s.buildConsignmentResponseDTO(ctx, &consignment, workflowNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, nil
}

// GetConsignmentsByTraderID retrieves consignments associated with a specific trader ID.
func (s *ConsignmentService) GetConsignmentsByTraderID(ctx context.Context, traderID string) ([]model.ConsignmentResponseDTO, error) {
	var consignments []model.Consignment
	result := s.db.WithContext(ctx).Where("trader_id = ?", traderID).Find(&consignments)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignments for trader %s: %w", traderID, result.Error)
	}

	if len(consignments) == 0 {
		return []model.ConsignmentResponseDTO{}, nil
	}

	// Extract all consignment IDs for batch query
	consignmentIDs := make([]uuid.UUID, len(consignments))
	for i, c := range consignments {
		consignmentIDs[i] = c.ID
	}

	// Fetch all workflow nodes in a single query to avoid N+1
	allWorkflowNodes, err := s.nodeRepo.GetWorkflowNodesByConsignmentIDsInTx(ctx, s.db, consignmentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for trader %s consignments: %w", traderID, err)
	}

	// Group workflow nodes by consignment ID for efficient lookup
	nodesByConsignment := make(map[uuid.UUID][]model.WorkflowNode)
	for _, node := range allWorkflowNodes {
		nodesByConsignment[node.ConsignmentID] = append(nodesByConsignment[node.ConsignmentID], node)
	}

	// Collect all HS Code IDs and Workflow Node Template IDs for batch fetching
	hsCodeIDsSet := make(map[uuid.UUID]struct{})
	for i := range consignments {
		for _, item := range consignments[i].Items {
			hsCodeIDsSet[item.HSCodeID] = struct{}{}
		}
	}
	templateIDsSet := make(map[uuid.UUID]struct{})
	for _, nodes := range nodesByConsignment {
		for _, node := range nodes {
			templateIDsSet[node.WorkflowNodeTemplateID] = struct{}{}
		}
	}

	// Batch fetch all required HS Codes
	hsCodeIDs := make([]uuid.UUID, 0, len(hsCodeIDsSet))
	for id := range hsCodeIDsSet {
		hsCodeIDs = append(hsCodeIDs, id)
	}
	var hsCodes []model.HSCode
	if len(hsCodeIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ?", hsCodeIDs).Find(&hsCodes).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch HS codes: %w", err)
		}
	}
	hsCodeMap := make(map[uuid.UUID]model.HSCode, len(hsCodes))
	for _, hsCode := range hsCodes {
		hsCodeMap[hsCode.ID] = hsCode
	}

	// Batch fetch all required Workflow Node Templates
	templateIDs := make([]uuid.UUID, 0, len(templateIDsSet))
	for id := range templateIDsSet {
		templateIDs = append(templateIDs, id)
	}
	var templates []model.WorkflowNodeTemplate
	if len(templateIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ?", templateIDs).Find(&templates).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch workflow node templates: %w", err)
		}
	}
	templateMap := make(map[uuid.UUID]model.WorkflowNodeTemplate, len(templates))
	for _, template := range templates {
		templateMap[template.ID] = template
	}

	// Build DTOs using pre-fetched data
	var consignmentDTOs []model.ConsignmentResponseDTO
	for i := range consignments {
		consignment := consignments[i]
		// Build item DTOs
		itemResponseDTOs := make([]model.ConsignmentItemResponseDTO, 0, len(consignment.Items))
		for _, item := range consignment.Items {
			hsCode, exists := hsCodeMap[item.HSCodeID]
			if !exists {
				return nil, fmt.Errorf("HS code not found for ID %s", item.HSCodeID)
			}
			itemResponseDTOs = append(itemResponseDTOs, model.ConsignmentItemResponseDTO{
				HSCode: model.HSCodeResponseDTO{
					HSCodeID:    hsCode.ID,
					HSCode:      hsCode.HSCode,
					Description: hsCode.Description,
					Category:    hsCode.Category,
				},
				ItemMetadata: item.ItemMetadata,
			})
		}

		// Build node DTOs
		workflowNodes := nodesByConsignment[consignment.ID]
		nodeResponseDTOs := make([]model.WorkflowNodeResponseDTO, 0, len(workflowNodes))
		for _, node := range workflowNodes {
			template, exists := templateMap[node.WorkflowNodeTemplateID]
			if !exists {
				return nil, fmt.Errorf("workflow node template not found for ID %s", node.WorkflowNodeTemplateID)
			}
			nodeResponseDTOs = append(nodeResponseDTOs, model.WorkflowNodeResponseDTO{
				ID:        node.ID,
				CreatedAt: node.CreatedAt.Format(time.RFC3339),
				UpdatedAt: node.UpdatedAt.Format(time.RFC3339),
				WorkflowNodeTemplate: model.WorkflowNodeTemplateResponseDTO{
					Name:        template.Name,
					Description: template.Description,
					Type:        string(template.Type),
				},
				State:     node.State,
				DependsOn: node.DependsOn,
			})
		}

		consignmentDTOs = append(consignmentDTOs, model.ConsignmentResponseDTO{
			ID:            consignment.ID,
			Flow:          consignment.Flow,
			TraderID:      consignment.TraderID,
			State:         consignment.State,
			Items:         itemResponseDTOs,
			CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
			WorkflowNodes: nodeResponseDTOs,
		})
	}

	return consignmentDTOs, nil
}

// UpdateConsignment updates an existing consignment in the database.
func (s *ConsignmentService) UpdateConsignment(ctx context.Context, updateReq *model.UpdateConsignmentDTO) (*model.ConsignmentResponseDTO, error) {
	if updateReq == nil {
		return nil, fmt.Errorf("update request cannot be nil")
	}

	var consignment model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", updateReq.ConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s for update: %w", updateReq.ConsignmentID, result.Error)
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
	workflowNodes, err := s.nodeRepo.GetWorkflowNodesByConsignmentIDInTx(ctx, s.db, consignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for consignment %s: %w", updateReq.ConsignmentID, err)
	}

	// Build response DTO using the helper function
	responseDTO, err := s.buildConsignmentResponseDTO(ctx, &consignment, workflowNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, nil
}

// UpdateWorkflowNodeStateAndPropagateChanges updates the state of a workflow node and propagates changes to dependent nodes and consignment state, returns the new READY nodes and newGlobalContext.
func (s *ConsignmentService) UpdateWorkflowNodeStateAndPropagateChanges(ctx context.Context, updateReq *model.UpdateWorkflowNodeDTO) ([]model.WorkflowNode, map[string]any, error) {
	if updateReq == nil {
		return nil, nil, fmt.Errorf("update request cannot be nil")
	}

	// Start a transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update the workflow node state and propagate changes, getting new READY nodes
	newReadyNodes, newGlobalContext, err := s.updateWorkflowNodeStateAndPropagateChangesInTx(ctx, tx, updateReq)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to update workflow node state and propagate changes: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newReadyNodes, newGlobalContext, nil
}

// updateWorkflowNodeStateAndPropagateChangesInTx updates the workflow node state and propagates changes within a transaction, and returns the new READY nodes.
func (s *ConsignmentService) updateWorkflowNodeStateAndPropagateChangesInTx(ctx context.Context, tx *gorm.DB, updateReq *model.UpdateWorkflowNodeDTO) ([]model.WorkflowNode, map[string]any, error) {
	// Get the workflow node
	workflowNode, err := s.nodeRepo.GetWorkflowNodeByIDInTx(ctx, tx, updateReq.WorkflowNodeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve workflow node with ID %s: %w", updateReq.WorkflowNodeID, err)
	}

	var newReadyNodes []model.WorkflowNode

	// Handle state transitions using the state machine
	switch updateReq.State {
	case model.WorkflowNodeStateFailed:
		if workflowNode.State != model.WorkflowNodeStateFailed {
			if err := s.stateMachine.TransitionToFailed(ctx, tx, workflowNode); err != nil {
				return nil, nil, fmt.Errorf("failed to transition node to FAILED: %w", err)
			}
		}

	case model.WorkflowNodeStateCompleted:
		if workflowNode.State != model.WorkflowNodeStateCompleted {
			result, err := s.stateMachine.TransitionToCompleted(ctx, tx, workflowNode)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to transition node to COMPLETED: %w", err)
			}
			newReadyNodes = result.NewReadyNodes

			// Update consignment state if all nodes are completed
			if result.AllNodesCompleted {
				if err := s.markConsignmentAsFinished(ctx, tx, workflowNode.ConsignmentID); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	// Handle global context updates
	var globalContext map[string]any
	if len(updateReq.AppendGlobalContext) > 0 {
		globalContext, err = s.appendToConsignmentGlobalContext(ctx, tx, workflowNode.ConsignmentID, updateReq.AppendGlobalContext)
		if err != nil {
			return nil, nil, err
		}
	}

	return newReadyNodes, globalContext, nil
}

// markConsignmentAsFinished updates the consignment state to FINISHED.
func (s *ConsignmentService) markConsignmentAsFinished(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) error {
	var consignment model.Consignment
	result := tx.WithContext(ctx).First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return fmt.Errorf("failed to retrieve consignment %s: %w", consignmentID, result.Error)
	}

	consignment.State = model.ConsignmentStateFinished
	if err := tx.WithContext(ctx).Save(&consignment).Error; err != nil {
		return fmt.Errorf("failed to update consignment %s state to FINISHED: %w", consignmentID, err)
	}

	return nil
}

// appendToConsignmentGlobalContext appends key-value pairs to the consignment's global context.
func (s *ConsignmentService) appendToConsignmentGlobalContext(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID, appendContext map[string]any) (map[string]any, error) {
	var consignment model.Consignment
	result := tx.WithContext(ctx).First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment %s: %w", consignmentID, result.Error)
	}

	if consignment.GlobalContext == nil {
		consignment.GlobalContext = make(map[string]any)
	}
	maps.Copy(consignment.GlobalContext, appendContext)

	if err := tx.WithContext(ctx).Save(&consignment).Error; err != nil {
		return nil, fmt.Errorf("failed to update consignment %s global context: %w", consignmentID, err)
	}

	return consignment.GlobalContext, nil
}

// buildConsignmentResponseDTO builds a ConsignmentResponseDTO from a Consignment and its WorkflowNodes
func (s *ConsignmentService) buildConsignmentResponseDTO(ctx context.Context, consignment *model.Consignment, workflowNodes []model.WorkflowNode) (*model.ConsignmentResponseDTO, error) {
	// Collect unique HS Code IDs from consignment items
	hsCodeIDsSet := make(map[uuid.UUID]struct{})
	for _, item := range consignment.Items {
		hsCodeIDsSet[item.HSCodeID] = struct{}{}
	}
	hsCodeIDs := make([]uuid.UUID, 0, len(hsCodeIDsSet))
	for id := range hsCodeIDsSet {
		hsCodeIDs = append(hsCodeIDs, id)
	}

	// Fetch HS Codes in a single query
	var hsCodes []model.HSCode
	if len(hsCodeIDs) > 0 {
		result := s.db.WithContext(ctx).Where("id IN ?", hsCodeIDs).Find(&hsCodes)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to fetch HS codes: %w", result.Error)
		}
	}

	// Create a map for quick HS Code lookup
	hsCodeMap := make(map[uuid.UUID]model.HSCode)
	for _, hsCode := range hsCodes {
		hsCodeMap[hsCode.ID] = hsCode
	}

	// Build ConsignmentItemResponseDTOs
	itemResponseDTOs := make([]model.ConsignmentItemResponseDTO, 0, len(consignment.Items))
	for _, item := range consignment.Items {
		hsCode, exists := hsCodeMap[item.HSCodeID]
		if !exists {
			return nil, fmt.Errorf("HS code not found for ID %s", item.HSCodeID)
		}

		itemResponseDTO := model.ConsignmentItemResponseDTO{
			HSCode: model.HSCodeResponseDTO{
				HSCodeID:    hsCode.ID,
				HSCode:      hsCode.HSCode,
				Description: hsCode.Description,
				Category:    hsCode.Category,
			},
			ItemMetadata: item.ItemMetadata,
		}
		itemResponseDTOs = append(itemResponseDTOs, itemResponseDTO)
	}

	// Collect unique WorkflowNodeTemplate IDs from workflow nodes
	templateIDsSet := make(map[uuid.UUID]struct{})
	for _, node := range workflowNodes {
		templateIDsSet[node.WorkflowNodeTemplateID] = struct{}{}
	}
	templateIDs := make([]uuid.UUID, 0, len(templateIDsSet))
	for id := range templateIDsSet {
		templateIDs = append(templateIDs, id)
	}

	// Fetch WorkflowNodeTemplates in a single query
	var templates []model.WorkflowNodeTemplate
	if len(templateIDs) > 0 {
		result := s.db.WithContext(ctx).Where("id IN ?", templateIDs).Find(&templates)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to fetch workflow node templates: %w", result.Error)
		}
	}

	// Create a map for quick template lookup
	templateMap := make(map[uuid.UUID]model.WorkflowNodeTemplate)
	for _, template := range templates {
		templateMap[template.ID] = template
	}

	// Build WorkflowNodeResponseDTOs
	nodeResponseDTOs := make([]model.WorkflowNodeResponseDTO, 0, len(workflowNodes))
	for _, node := range workflowNodes {
		template, exists := templateMap[node.WorkflowNodeTemplateID]
		if !exists {
			return nil, fmt.Errorf("workflow node template not found for ID %s", node.WorkflowNodeTemplateID)
		}

		// Use name and description directly from the template
		nodeResponseDTO := model.WorkflowNodeResponseDTO{
			ID:        node.ID,
			CreatedAt: node.CreatedAt.Format(time.RFC3339),
			UpdatedAt: node.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeTemplate: model.WorkflowNodeTemplateResponseDTO{
				Name:        template.Name,
				Description: template.Description,
				Type:        string(template.Type),
			},
			State:     node.State,
			DependsOn: node.DependsOn,
		}
		nodeResponseDTOs = append(nodeResponseDTOs, nodeResponseDTO)
	}

	// Build the final ConsignmentResponseDTO
	responseDTO := &model.ConsignmentResponseDTO{
		ID:            consignment.ID,
		Flow:          consignment.Flow,
		TraderID:      consignment.TraderID,
		State:         consignment.State,
		Items:         itemResponseDTOs,
		CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes: nodeResponseDTOs,
	}

	return responseDTO, nil
}
