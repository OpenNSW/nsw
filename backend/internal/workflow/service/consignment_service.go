package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"gorm.io/gorm"

	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"

	workflowmanagerV1 "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/utils"
)

// ConsignmentService handles consignment-related operations.
// It coordinates between workflow templates, nodes, and the workflow manager.
// It also implements WorkflowEventHandler for domain-specific lifecycle callbacks.
type ConsignmentService struct {
	db                *gorm.DB
	templateProvider  TemplateProvider
	workflowManagerV1 workflowmanagerV1.Manager
	workflowManagerV2 workflowManagerV2.Manager
}

// NewConsignmentService creates a new instance of ConsignmentService.
func NewConsignmentService(db *gorm.DB, templateProvider TemplateProvider, wmV1 workflowmanagerV1.Manager, wmV2 workflowManagerV2.Manager) *ConsignmentService {
	return &ConsignmentService{
		db:                db,
		templateProvider:  templateProvider,
		workflowManagerV1: wmV1,
		workflowManagerV2: wmV2,
	}
}

// --- WorkflowEventHandler implementation ---

// OnWorkflowStatusChanged handles workflow lifecycle state propagation to consignment domain state.
func (s *ConsignmentService) OnWorkflowStatusChanged(_ context.Context, tx *gorm.DB, workflowID string, _ model.WorkflowStatus, toStatus model.WorkflowStatus, _ *model.Workflow) error {
	switch toStatus {
	case model.WorkflowStatusCompleted:
		return s.markConsignmentAsFinished(tx, workflowID)
	default:
		return nil
	}
}

// CreateConsignmentShell creates a shell consignment (Stage 1: Trader selects CHA). State is INITIALIZED; no workflow nodes.
func (s *ConsignmentService) CreateConsignmentShell(ctx context.Context, flow model.ConsignmentFlow, chaID string, traderID string) (*model.ConsignmentDetailDTO, error) {
	if traderID == "" {
		return nil, fmt.Errorf("trader ID cannot be empty")
	}
	// Validate CHA exists
	var cha model.CHA
	if err := s.db.WithContext(ctx).First(&cha, "id = ?", chaID).Error; err != nil {
		return nil, fmt.Errorf("CHA not found: %w", err)
	}
	consignment := &model.Consignment{
		Flow:     flow,
		TraderID: traderID,
		CHAID:    chaID,
		State:    model.ConsignmentStateInitialized,
		Items:    []model.ConsignmentItem{},
	}
	if err := s.db.WithContext(ctx).Create(consignment).Error; err != nil {
		return nil, fmt.Errorf("failed to create consignment: %w", err)
	}
	// Reload for response (no workflow nodes at stage 1)
	if err := s.db.WithContext(ctx).First(consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment: %w", err)
	}
	responseDTO, err := s.buildConsignmentDetailDTO(ctx, consignment, nil, nil, newHSCodeBatchLoader(s.db))
	if err != nil {
		return nil, err
	}
	return responseDTO, nil
}

// InitializeConsignmentByID runs Stage 2: CHA selects one or more HS Codes; creates workflow and sets state to IN_PROGRESS.
// Returns error if consignment is not in INITIALIZED.
func (s *ConsignmentService) InitializeConsignmentByID(
	ctx context.Context,
	consignmentID string,
	hsCodeIDs []string,
	globalContext map[string]any,
) (*model.ConsignmentDetailDTO, error) {

	if len(hsCodeIDs) == 0 {
		return nil, fmt.Errorf("at least one HS code ID is required")
	}

	var consignment model.Consignment
	if err := s.db.WithContext(ctx).First(&consignment, "id = ?", consignmentID).Error; err != nil {
		return nil, fmt.Errorf("consignment not found: %w", err)
	}

	if consignment.State != model.ConsignmentStateInitialized {
		return nil, fmt.Errorf("consignment must be in INITIALIZED (current state: %s)", consignment.State)
	}

	// Prepare items (common for both V1 and V2)
	items := make([]model.ConsignmentItem, 0, len(hsCodeIDs))
	for _, hsCodeID := range hsCodeIDs {
		items = append(items, model.ConsignmentItem{HSCodeID: hsCodeID})
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	consignment.Items = items
	consignment.State = model.ConsignmentStateInProgress

	if err := tx.Save(&consignment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update consignment: %w", err)
	}

	if s.workflowManagerV2 != nil {
		// TODO: add support for collapsing multiple HS codes to one workflow.
		// Currently, assumes that there is only one HS code selected.
		if len(hsCodeIDs) > 1 {
			tx.Rollback()
			return nil, fmt.Errorf("v2 currently supports only one HS code")
		}

		wtV2, err := s.templateProvider.GetWorkflowTemplateByHSCodeIDAndFlowV2(ctx, hsCodeIDs[0], consignment.Flow)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to get v2 workflow template: %w", err)
		}
		if wtV2 == nil {
			tx.Rollback()
			return nil, fmt.Errorf("no v2 workflow template found for HS code %s and flow %s", hsCodeIDs[0], consignment.Flow)
		}

		if err := s.workflowManagerV2.StartWorkflow(ctx, consignment.ID, wtV2.WorkflowDefinition, globalContext); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to register workflow: %w", err)
		}

	} else {
		var workflowTemplates []model.WorkflowTemplate

		for _, hsCodeID := range hsCodeIDs {
			wt, err := s.templateProvider.GetWorkflowTemplateByHSCodeIDAndFlow(ctx, hsCodeID, consignment.Flow)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to get workflow template for HS code %s and flow %s: %w", hsCodeID, consignment.Flow, err)
			}
			if wt == nil {
				tx.Rollback()
				return nil, fmt.Errorf("no workflow template found for HS code %s and flow %s", hsCodeID, consignment.Flow)
			}

			workflowTemplates = append(workflowTemplates, *wt)
		}

		if err := s.workflowManagerV1.StartWorkflowInstance(ctx, tx, consignment.ID, workflowTemplates, globalContext, s); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to register workflow: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Reload for response
	if err := s.db.WithContext(ctx).First(&consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment: %w", err)
	}

	var wf *model.Workflow
	var twf *workflowManagerV2.WorkflowInstance

	if s.workflowManagerV2 != nil {
		var err error
		twf, err = s.workflowManagerV2.GetStatus(ctx, consignment.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get v2 workflow details: %w", err)
		}
	} else {
		var err error
		wf, err = s.workflowManagerV1.GetWorkflowInstance(ctx, consignment.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get legacy workflow details: %w", err)
		}
	}

	hsLoader := newHSCodeBatchLoader(s.db)
	hsLoader.collectFromItems(consignment.Items)

	if err := hsLoader.load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, wf, twf, hsLoader)
	if err != nil {
		return nil, err
	}

	return responseDTO, nil
}

// GetConsignmentByID retrieves a consignment by its ID from the database.
func (s *ConsignmentService) GetConsignmentByID(ctx context.Context, consignmentID string) (*model.ConsignmentDetailDTO, error) {
	var consignment model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s: %w", consignmentID, result.Error)
	}

	// Load workflow details (nodes + templates) if workflow exists
	var wf *model.Workflow
	var twf *workflowManagerV2.WorkflowInstance
	if consignment.State != model.ConsignmentStateInitialized {
		if s.workflowManagerV2 != nil {
			var err error
			twf, err = s.workflowManagerV2.GetStatus(ctx, consignment.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get v2 workflow details: %w", err)
			}
		} else {
			var err error
			wf, err = s.workflowManagerV1.GetWorkflowInstance(ctx, consignment.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get workflow details: %w", err)
			}
		}
	}

	hsLoader := newHSCodeBatchLoader(s.db)
	hsLoader.collectFromItems(consignment.Items)
	if err := hsLoader.load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, wf, twf, hsLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, nil
}

// ListConsignments returns consignments filtered by trader (role=trader) or by CHA (role=cha). Exactly one of filter.TraderID or filter.ChaID must be set.
func (s *ConsignmentService) ListConsignments(ctx context.Context, filter model.ConsignmentFilter) (*model.ConsignmentListResult, error) {
	var baseQuery *gorm.DB
	if filter.ChaID != nil {
		baseQuery = s.db.WithContext(ctx).Model(&model.Consignment{}).Where("cha_id = ?", *filter.ChaID)
	} else if filter.TraderID != nil {
		baseQuery = s.db.WithContext(ctx).Model(&model.Consignment{}).Where("trader_id = ?", *filter.TraderID)
	} else {
		return nil, fmt.Errorf("either TraderID or ChaID must be set in filter")
	}
	return s.listConsignmentsWithBaseQuery(ctx, baseQuery, filter)
}

// GetConsignmentsByTraderID retrieves consignments associated with a specific trader ID with optional filtering.
func (s *ConsignmentService) GetConsignmentsByTraderID(ctx context.Context, traderID string, offset *int, limit *int, filter model.ConsignmentFilter) (*model.ConsignmentListResult, error) {
	filter.TraderID = &traderID
	filter.Offset = offset
	filter.Limit = limit
	return s.ListConsignments(ctx, filter)
}

// listConsignmentsWithBaseQuery runs the shared list logic (filters, count, pagination, DTOs).
func (s *ConsignmentService) listConsignmentsWithBaseQuery(ctx context.Context, baseQuery *gorm.DB, filter model.ConsignmentFilter) (*model.ConsignmentListResult, error) {
	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(filter.Offset, filter.Limit)

	// Apply optional filters
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
	consignmentIDs := make([]string, len(consignments))
	for i, c := range consignments {
		consignmentIDs[i] = c.ID
	}

	// Fetch workflow node counts in batch (via workflow_id which equals consignment ID)
	type NodeCounts struct {
		WorkflowID string
		Total      int
		Completed  int
	}

	var nodeCounts []NodeCounts
	err := s.db.WithContext(ctx).Model(&model.WorkflowNode{}).
		Select("workflow_id, count(*) as total, count(case when state = ? then 1 end) as completed", model.WorkflowNodeStateCompleted).
		Where("workflow_id IN ?", consignmentIDs).
		Group("workflow_id").
		Scan(&nodeCounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow node counts: %w", err)
	}

	// Map counts to consignment IDs (workflow_id == consignment_id) for easy lookup
	countsMap := make(map[string]NodeCounts)
	for _, nc := range nodeCounts {
		countsMap[nc.WorkflowID] = nc
	}

	// Check which consignments have end nodes (via the workflows table)
	type WorkflowEndNode struct {
		ID        string
		EndNodeID *string
	}
	var workflowEndNodes []WorkflowEndNode
	err = s.db.WithContext(ctx).Model(&model.Workflow{}).
		Select("id, end_node_id").
		Where("id IN ?", consignmentIDs).
		Scan(&workflowEndNodes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow end nodes: %w", err)
	}
	endNodeMap := make(map[string]bool)
	for _, w := range workflowEndNodes {
		if w.EndNodeID != nil {
			endNodeMap[w.ID] = true
		}
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

		// If the workflow has an EndNode, subtract it from the total count
		// (since it's an internal implementation detail not shown to users)
		if endNodeMap[c.ID] {
			if counts.Total > 0 {
				counts.Total -= 1
			}
		}

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
			ChaID:                      c.CHAID,
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
// TODO: clean up? this doesn't seem to be used anywhere other than tests?
func (s *ConsignmentService) UpdateConsignment(ctx context.Context, updateReq *model.UpdateConsignmentDTO) (*model.ConsignmentDetailDTO, error) {
	if updateReq == nil {
		return nil, fmt.Errorf("update request cannot be nil")
	}

	var consignment model.Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", updateReq.ConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s for update: %w", updateReq.ConsignmentID, result.Error)
	}

	// Build updates map to only update changed fields
	updates := make(map[string]interface{})

	if updateReq.State != nil {
		updates["state"] = *updateReq.State
	}

	// Use Updates to only modify specified fields
	if len(updates) > 0 {
		saveResult := s.db.WithContext(ctx).Model(&consignment).Updates(updates)
		if saveResult.Error != nil {
			return nil, fmt.Errorf("failed to update consignment: %w", saveResult.Error)
		}
	}

	// Handle global context updates via the Workflow entity
	if updateReq.AppendToGlobalContext != nil {
		tx := s.db.WithContext(ctx).Begin()
		var wf model.Workflow
		if err := tx.First(&wf, "id = ?", updateReq.ConsignmentID).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to retrieve workflow for consignment %s: %w", updateReq.ConsignmentID, err)
		}
		if wf.GlobalContext == nil {
			wf.GlobalContext = make(map[string]any)
		}
		maps.Copy(wf.GlobalContext, updateReq.AppendToGlobalContext)
		if err := tx.Save(&wf).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update workflow global context: %w", err)
		}
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to commit global context update: %w", err)
		}
	}

	// Reload for response
	if err := s.db.WithContext(ctx).First(&consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment: %w", err)
	}

	var wf *model.Workflow
	var twf *workflowManagerV2.WorkflowInstance
	if consignment.State != model.ConsignmentStateInitialized {
		if s.workflowManagerV2 != nil {
			var err error
			twf, err = s.workflowManagerV2.GetStatus(ctx, consignment.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get v2 workflow details: %w", err)
			}
		} else {
			var err error
			wf, err = s.workflowManagerV1.GetWorkflowInstance(ctx, consignment.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get workflow details: %w", err)
			}
		}
	}

	hsLoader := newHSCodeBatchLoader(s.db)
	hsLoader.collectFromItems(consignment.Items)
	if err := hsLoader.load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load HS codes: %w", err)
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, wf, twf, hsLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, nil
}

// markConsignmentAsFinished updates the consignment state to FINISHED.
func (s *ConsignmentService) markConsignmentAsFinished(tx *gorm.DB, consignmentID string) error {
	var consignment model.Consignment
	if err := tx.First(&consignment, "id = ?", consignmentID).Error; err != nil {
		return fmt.Errorf("failed to retrieve consignment %s: %w", consignmentID, err)
	}
	consignment.State = model.ConsignmentStateFinished
	if err := tx.Save(&consignment).Error; err != nil {
		return fmt.Errorf("failed to update consignment %s state to FINISHED: %w", consignmentID, err)
	}
	return nil
}

// hsCodeBatchLoader handles batch loading of HS codes for JSONB items
type hsCodeBatchLoader struct {
	db        *gorm.DB
	hsCodeMap map[string]model.HSCode
	hsCodeIDs map[string]struct{}
}

func newHSCodeBatchLoader(db *gorm.DB) *hsCodeBatchLoader {
	return &hsCodeBatchLoader{
		db:        db,
		hsCodeMap: make(map[string]model.HSCode),
		hsCodeIDs: make(map[string]struct{}),
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

	hsCodeIDList := make([]string, 0, len(loader.hsCodeIDs))
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

func (loader *hsCodeBatchLoader) get(id string) (model.HSCode, error) {
	hsCode, exists := loader.hsCodeMap[id]
	if !exists {
		return model.HSCode{}, fmt.Errorf("HS code not found for ID %s", id)
	}
	return hsCode, nil
}

// buildConsignmentDetailDTO builds a ConsignmentDetailDTO from a Consignment.
// The workflow parameter provides the workflow nodes (nil for INITIALIZED consignments).
func (s *ConsignmentService) buildConsignmentDetailDTO(
	ctx context.Context,
	consignment *model.Consignment,
	workflowV1 *model.Workflow,
	workflowV2 *workflowManagerV2.WorkflowInstance,
	hsLoader *hsCodeBatchLoader,
) (*model.ConsignmentDetailDTO, error) {
	itemResponseDTOs, err := s.buildConsignmentItemResponseDTOs(consignment.Items, hsLoader)
	if err != nil {
		return nil, err
	}

	nodeResponseDTOs := make([]model.WorkflowNodeResponseDTO, 0)
	edgeResponseDTOs := make([]model.WorkflowEdgeResponseDTO, 0)

	if workflowV1 != nil {
		nodeResponseDTOs = make([]model.WorkflowNodeResponseDTO, 0, len(workflowV1.WorkflowNodes))
		for _, node := range workflowV1.WorkflowNodes {
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
				Outcome:       node.Outcome,
				DependsOn:     node.DependsOn,
			})
		}
	} else if workflowV2 != nil {
		taskTemplateIDs := make([]string, 0, len(workflowV2.NodeInfo))
		for _, node := range workflowV2.NodeInfo {
			if node.Type == workflowManagerV2.NodeTypeTask {
				taskTemplateIDs = append(taskTemplateIDs, node.TaskTemplateID)
			}
		}
		taskTemplates, err := s.templateProvider.GetWorkflowNodeTemplatesByIDs(ctx, taskTemplateIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve workflow node templates for consignment %s: %w", consignment.ID, err)
		}
		taskTemplateMap := make(map[string]model.WorkflowNodeTemplate)
		for _, taskTemplate := range taskTemplates {
			taskTemplateMap[taskTemplate.ID] = taskTemplate
		}
		nodeResponseDTOs = make([]model.WorkflowNodeResponseDTO, 0, len(workflowV2.NodeInfo))
		for _, node := range workflowV2.NodeInfo {
			var taskName, taskDescription, taskType string
			var nodeState model.WorkflowNodeState
			if node.Type == workflowManagerV2.NodeTypeTask {
				taskTemplate, ok := taskTemplateMap[node.TaskTemplateID]
				if !ok {
					slog.Error("failed to retrieve workflow node template for", "consignment_id", consignment.ID, "node_id", node.ID, "task_template_id", node.TaskTemplateID)
					return nil, fmt.Errorf("failed to retrieve workflow node template %s for node %s", node.TaskTemplateID, node.ID)
				}
				taskName = taskTemplate.Name
				taskDescription = taskTemplate.Description
				taskType = string(taskTemplate.Type)
			} else {
				taskType = string(node.Type)
			}
			// TODO: clean up translations once the frontend is updated.
			switch node.Status {
			case workflowManagerV2.NodeStatusRunning:
				nodeState = model.WorkflowNodeStateInProgress
			case workflowManagerV2.NodeStatusCompleted:
				nodeState = model.WorkflowNodeStateCompleted
			case workflowManagerV2.NodeStatusFailed:
				nodeState = model.WorkflowNodeStateFailed
			case workflowManagerV2.NodeStatusNotStarted:
				nodeState = model.WorkflowNodeStateLocked
			}
			nodeResponseDTOs = append(nodeResponseDTOs, model.WorkflowNodeResponseDTO{
				ID:        node.ID,
				CreatedAt: node.CreatedAt.Format(time.RFC3339),
				UpdatedAt: node.UpdatedAt.Format(time.RFC3339),
				WorkflowNodeTemplate: model.WorkflowNodeTemplateResponseDTO{
					Name:        taskName,
					Description: taskDescription,
					Type:        taskType,
				},
				State:     nodeState,
				DependsOn: []string{}, // TODO: should be removed or should be populated based on the workflow definition (not currently stored in DB for v2 workflows)
			})
		}
	}

	if workflowV2 != nil {
		for _, edge := range workflowV2.Edges {
			edgeResponseDTOs = append(edgeResponseDTOs, model.WorkflowEdgeResponseDTO{
				ID:        edge.ID,
				SourceID:  edge.SourceID,
				TargetID:  edge.TargetID,
				Condition: edge.Condition,
			})
		}
	}

	return &model.ConsignmentDetailDTO{
		ID:            consignment.ID,
		Flow:          consignment.Flow,
		TraderID:      consignment.TraderID,
		ChaID:         consignment.CHAID,
		State:         consignment.State,
		Items:         itemResponseDTOs,
		CreatedAt:     consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes: nodeResponseDTOs,
		Edges:         edgeResponseDTOs,
	}, nil
}

// buildConsignmentItemResponseDTOs builds a slice of ConsignmentItemResponseDTO from ConsignmentItems.
func (s *ConsignmentService) buildConsignmentItemResponseDTOs(items []model.ConsignmentItem, hsLoader *hsCodeBatchLoader) ([]model.ConsignmentItemResponseDTO, error) {
	itemResponseDTOs := make([]model.ConsignmentItemResponseDTO, 0, len(items))
	for _, item := range items {
		hsCode, err := hsLoader.get(item.HSCodeID)
		if err != nil {
			return nil, err
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
