package service

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/utils"
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
func (s *ConsignmentService) InitializeConsignment(ctx context.Context, createReq *model.CreateConsignmentDTO, traderId string, globalContext map[string]any) (*model.ConsignmentDetailDTO, []model.WorkflowNode, error) {
	if createReq == nil {
		return nil, nil, fmt.Errorf("create request cannot be nil")
	}
	if len(createReq.Items) == 0 {
		return nil, nil, fmt.Errorf("consignment must have at least one item")
	}
	if traderId == "" {
		return nil, nil, fmt.Errorf("trader ID cannot be empty")
	}

	consignment, newReadyWorkflowNodes, err := s.initializeConsignmentInTx(ctx, createReq, traderId, globalContext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize consignment: %w", err)
	}

	return consignment, newReadyWorkflowNodes, nil
}

// initializeConsignmentInTx initializes the consignment within a transaction.
func (s *ConsignmentService) initializeConsignmentInTx(ctx context.Context, createReq *model.CreateConsignmentDTO, traderId string, globalContext map[string]any) (*model.ConsignmentDetailDTO, []model.WorkflowNode, error) {
	consignment := &model.Consignment{
		Flow:          createReq.Flow,
		TraderID:      traderId,
		State:         model.ConsignmentStateInProgress,
		GlobalContext: globalContext,
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
	_, newReadyWorkflowNodes, err := s.createWorkflowNodesInTx(ctx, tx, consignment.ID, workflowTemplates)
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

	// Reload consignment with preloaded relationships for response building
	if err := s.db.WithContext(ctx).Preload("WorkflowNodes.WorkflowNodeTemplate").First(consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to reload consignment with relationships: %w", err)
	}

	// Prepare Response DTO using the helper function
	hsLoader := newHSCodeBatchLoader(s.db)
	hsLoader.collectFromItems(consignment.Items)
	if err := hsLoader.load(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, consignment, hsLoader)
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
	return s.stateMachine.InitializeNodesFromTemplates(ctx, tx, ParentRef{ConsignmentID: &consignmentID}, nodeTemplates)
}

// GetConsignmentByID retrieves a consignment by its ID from the database.
func (s *ConsignmentService) GetConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*model.ConsignmentDetailDTO, error) {
	var consignment model.Consignment
	// Use Preload to fetch WorkflowNodes and their templates in a single query
	result := s.db.WithContext(ctx).Preload("WorkflowNodes.WorkflowNodeTemplate").First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s: %w", consignmentID, result.Error)
	}

	// Batch load HS codes for JSONB items
	hsLoader := newHSCodeBatchLoader(s.db)
	hsLoader.collectFromItems(consignment.Items)
	if err := hsLoader.load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	// Build response DTO using the helper function
	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, hsLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, nil
}

// GetConsignmentsByTraderID retrieves consignments associated with a specific trader ID with optional filtering.
func (s *ConsignmentService) GetConsignmentsByTraderID(ctx context.Context, traderID string, offset *int, limit *int, filter model.ConsignmentFilter) (*model.ConsignmentListResult, error) {
	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(offset, limit)

	// Base query for this trader
	baseQuery := s.db.WithContext(ctx).Model(&model.Consignment{}).Where("trader_id = ?", traderID)

	// Apply Filters
	query := baseQuery
	if filter.State != nil {
		query = query.Where("state = ?", *filter.State)
	}
	if filter.Flow != nil {
		query = query.Where("flow = ?", *filter.Flow)
	}

	// Get total count of FILTERED records
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count filtered consignments: %w", err)
	}

	if totalCount == 0 {
		return &model.ConsignmentListResult{
			TotalCount: 0,
			Items:      []model.ConsignmentSummaryDTO{},
			Offset:     finalOffset,
			Limit:      finalLimit,
		}, nil
	}

	var consignments []model.Consignment
	// Apply Pagination and Ordering to the filtered query
	// NOTE: We do NOT preload WorkflowNodes here to improve performance
	query = query.
		Offset(finalOffset).
		Limit(finalLimit).
		Order("created_at DESC")

	if err := query.Find(&consignments).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve consignments: %w", err)
	}

	// Collect Consignment IDs to fetch workflow node counts
	consignmentIDs := make([]uuid.UUID, len(consignments))
	for i, c := range consignments {
		consignmentIDs[i] = c.ID
	}

	// Fetch workflow node counts in batch
	// We need counts of ALL nodes and COMPLETED nodes per consignment
	type NodeCounts struct {
		ConsignmentID uuid.UUID
		Total         int
		Completed     int
	}

	var nodeCounts []NodeCounts
	// This query groups by consignment_id and counts total and completed nodes
	// It assumes workflow_nodes table has a consignment_id column and state column
	err := s.db.WithContext(ctx).Model(&model.WorkflowNode{}).
		Select("consignment_id, count(*) as total, count(case when state = ? then 1 end) as completed", model.WorkflowNodeStateCompleted).
		Where("consignment_id IN ?", consignmentIDs).
		Group("consignment_id").
		Scan(&nodeCounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow node counts: %w", err)
	}

	// Map counts to consignment IDs for easy lookup
	countsMap := make(map[uuid.UUID]NodeCounts)
	for _, nc := range nodeCounts {
		countsMap[nc.ConsignmentID] = nc
	}

	// Batch load HS codes for all JSONB items
	hsLoader := newHSCodeBatchLoader(s.db)
	for i := range consignments {
		hsLoader.collectFromItems(consignments[i].Items)
	}
	if err := hsLoader.load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	// Build Summary DTOs for all consignments
	var consignmentDTOs []model.ConsignmentSummaryDTO
	for i := range consignments {
		c := consignments[i]
		counts := countsMap[c.ID]

		// Build Item Response DTOs
		itemResponseDTOs, err := s.buildConsignmentItemResponseDTOs(c.Items, hsLoader)
		if err != nil {
			// Handle error gracefully, or log it. For now, failing the whole request might be too harsh if just one HS code is missing (data inconsistency)
			// returning error for now to be safe
			return nil, fmt.Errorf("failed to load HS code for item in consignment %s: %w", c.ID, err)
		}

		consignmentDTOs = append(consignmentDTOs, model.ConsignmentSummaryDTO{
			ID:                         c.ID,
			Flow:                       c.Flow,
			TraderID:                   c.TraderID,
			State:                      c.State,
			Items:                      itemResponseDTOs,
			CreatedAt:                  c.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                  c.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeCount:          counts.Total,
			CompletedWorkflowNodeCount: counts.Completed,
		})
	}

	return &model.ConsignmentListResult{
		TotalCount: totalCount,
		Items:      consignmentDTOs,
		Offset:     finalOffset,
		Limit:      finalLimit,
	}, nil
}

// UpdateConsignment updates an existing consignment in the database.
func (s *ConsignmentService) UpdateConsignment(ctx context.Context, updateReq *model.UpdateConsignmentDTO) (*model.ConsignmentDetailDTO, error) {
	if updateReq == nil {
		return nil, fmt.Errorf("update request cannot be nil")
	}

	var consignment model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", updateReq.ConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s for update: %w", updateReq.ConsignmentID, result.Error)
	}

	// Build updates map to only update changed fields (avoids overwriting concurrent changes)
	updates := make(map[string]interface{})

	if updateReq.State != nil {
		updates["state"] = *updateReq.State
	}

	if updateReq.AppendToGlobalContext != nil {
		// For GlobalContext, we need to merge carefully
		if consignment.GlobalContext == nil {
			consignment.GlobalContext = make(map[string]any)
		}
		maps.Copy(consignment.GlobalContext, updateReq.AppendToGlobalContext)
		updates["global_context"] = consignment.GlobalContext
		// TODO: Implement the global context key selection such that no overwriting occurs.
	}

	// Use Updates to only modify specified fields, reducing race condition risk
	if len(updates) > 0 {
		saveResult := s.db.WithContext(ctx).Model(&consignment).Updates(updates)
		if saveResult.Error != nil {
			return nil, fmt.Errorf("failed to update consignment: %w", saveResult.Error)
		}
	}

	// Reload with preloaded relationships
	if err := s.db.WithContext(ctx).Preload("WorkflowNodes.WorkflowNodeTemplate").First(&consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment with relationships: %w", err)
	}

	// Batch load HS codes for JSONB items
	hsLoader := newHSCodeBatchLoader(s.db)
	hsLoader.collectFromItems(consignment.Items)
	if err := hsLoader.load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	// Build response DTO using the helper function
	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, hsLoader)
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
			if err := s.stateMachine.TransitionToFailed(ctx, tx, workflowNode, updateReq); err != nil {
				return nil, nil, fmt.Errorf("failed to transition node to FAILED: %w", err)
			}
		}

	case model.WorkflowNodeStateInProgress:
		if err := s.stateMachine.TransitionToInProgress(ctx, tx, workflowNode, updateReq); err != nil {
			return nil, nil, fmt.Errorf("failed to transition node to IN_PROGRESS: %w", err)
		}

	case model.WorkflowNodeStateCompleted:
		if workflowNode.State != model.WorkflowNodeStateCompleted {
			result, err := s.stateMachine.TransitionToCompleted(ctx, tx, workflowNode, updateReq)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to transition node to COMPLETED: %w", err)
			}
			newReadyNodes = result.NewReadyNodes

			// Update consignment state if all nodes are completed
			if result.AllNodesCompleted {
				if err := s.markConsignmentAsFinished(ctx, tx, *workflowNode.ConsignmentID); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	// Handle global context updates
	var globalContext map[string]any
	globalContext, err = s.appendToConsignmentGlobalContext(ctx, tx, *workflowNode.ConsignmentID, updateReq.AppendGlobalContext)
	if err != nil {
		return nil, nil, err
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

// hsCodeBatchLoader handles batch loading of HS codes for JSONB items
type hsCodeBatchLoader struct {
	db        *gorm.DB
	hsCodeMap map[uuid.UUID]model.HSCode
	hsCodeIDs map[uuid.UUID]struct{}
}

func newHSCodeBatchLoader(db *gorm.DB) *hsCodeBatchLoader {
	return &hsCodeBatchLoader{
		db:        db,
		hsCodeMap: make(map[uuid.UUID]model.HSCode),
		hsCodeIDs: make(map[uuid.UUID]struct{}),
	}
}

func (loader *hsCodeBatchLoader) collectFromItems(items []model.ConsignmentItem) {
	for _, item := range items {
		loader.hsCodeIDs[item.HSCodeID] = struct{}{}
	}
}

func (loader *hsCodeBatchLoader) load(ctx context.Context) error {
	if len(loader.hsCodeIDs) == 0 {
		return nil
	}

	hsCodeIDList := make([]uuid.UUID, 0, len(loader.hsCodeIDs))
	for id := range loader.hsCodeIDs {
		hsCodeIDList = append(hsCodeIDList, id)
	}

	var hsCodes []model.HSCode
	if err := loader.db.WithContext(ctx).Where("id IN ?", hsCodeIDList).Find(&hsCodes).Error; err != nil {
		return fmt.Errorf("failed to batch load HS codes: %w", err)
	}

	for _, hsCode := range hsCodes {
		loader.hsCodeMap[hsCode.ID] = hsCode
	}

	return nil
}

func (loader *hsCodeBatchLoader) get(id uuid.UUID) (model.HSCode, error) {
	hsCode, exists := loader.hsCodeMap[id]
	if !exists {
		return model.HSCode{}, fmt.Errorf("HS code not found for ID %s", id)
	}
	return hsCode, nil
}

// buildConsignmentDetailDTO builds a ConsignmentDetailDTO from a Consignment with preloaded WorkflowNodes
func (s *ConsignmentService) buildConsignmentDetailDTO(_ context.Context, consignment *model.Consignment, hsLoader *hsCodeBatchLoader) (*model.ConsignmentDetailDTO, error) {
	// Build ConsignmentItemResponseDTOs using the batch loader
	// Build ConsignmentItemResponseDTOs using the batch loader
	itemResponseDTOs, err := s.buildConsignmentItemResponseDTOs(consignment.Items, hsLoader)
	if err != nil {
		return nil, err
	}

	// Build WorkflowNodeResponseDTOs using preloaded templates
	nodeResponseDTOs := make([]model.WorkflowNodeResponseDTO, 0, len(consignment.WorkflowNodes))
	for _, node := range consignment.WorkflowNodes {
		nodeResponseDTOs = append(nodeResponseDTOs, model.WorkflowNodeResponseDTO{
			ID:        node.ID,
			CreatedAt: node.CreatedAt.Format(time.RFC3339),
			UpdatedAt: node.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeTemplate: model.WorkflowNodeTemplateResponseDTO{
				Name:        node.WorkflowNodeTemplate.Name,
				Description: node.WorkflowNodeTemplate.Description,
				Type:        string(node.WorkflowNodeTemplate.Type),
			},
			State:         node.State,
			ExtendedState: node.ExtendedState,
			DependsOn:     node.DependsOn,
		})
	}

	// Build the final ConsignmentDetailDTO
	responseDTO := &model.ConsignmentDetailDTO{
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

// buildConsignmentItemResponseDTOs builds a slice of ConsignmentItemResponseDTO from ConsignmentItems.
func (s *ConsignmentService) buildConsignmentItemResponseDTOs(items []model.ConsignmentItem, hsLoader *hsCodeBatchLoader) ([]model.ConsignmentItemResponseDTO, error) {
	itemResponseDTOs := make([]model.ConsignmentItemResponseDTO, 0, len(items))
	for _, item := range items {
		hsCode, err := hsLoader.get(item.HSCodeID)
		if err != nil {
			return nil, err // The caller can wrap the error with more context if needed.
		}
		itemResponseDTOs = append(itemResponseDTOs, model.ConsignmentItemResponseDTO{
			HSCode: model.HSCodeResponseDTO{
				HSCodeID:    hsCode.ID,
				HSCode:      hsCode.HSCode,
				Description: hsCode.Description,
				Category:    hsCode.Category,
			},
		})
	}
	return itemResponseDTOs, nil
}
