package consignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	workflowmanager "github.com/OpenNSW/go-temporal-workflow"

	"github.com/OpenNSW/nsw/backend/internal/hscode"
	"github.com/OpenNSW/nsw/backend/internal/profile/cha"
	"github.com/OpenNSW/nsw/backend/internal/profile/company"
	"github.com/OpenNSW/nsw/backend/internal/profile/user"
	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
	"github.com/OpenNSW/nsw/backend/internal/workflow/service"
	"github.com/OpenNSW/nsw/backend/utils"
)

// Service handles consignment-related operations.
// It coordinates between workflow templates, nodes, and the workflow manager.
// It also implements WorkflowEventHandler for domain-specific lifecycle callbacks.
// TODO: Clean this up to use TemplateService directly instead of the TemplateProvider interface
// once the database-backed template setup is completely retired.
type Service struct {
	db               *gorm.DB
	templateProvider service.TemplateProvider
	wm               workflowmanager.Manager
	chaService       cha.Service
	companyService   company.Service
	userService      user.Service
	hsCodeService    *hscode.Service
}

// NewService creates a new instance of Service.
func NewService(
	db *gorm.DB,
	templateProvider service.TemplateProvider,
	chaService cha.Service,
	companyService company.Service,
	userService user.Service,
	hsCodeService *hscode.Service,
) *Service {
	return &Service{
		db:               db,
		templateProvider: templateProvider,
		chaService:       chaService,
		companyService:   companyService,
		userService:      userService,
		hsCodeService:    hsCodeService,
	}
}

// RegisterWorkflowManager registers the workflow manager
func (s *Service) RegisterWorkflowManager(wm workflowmanager.Manager) error {
	if s.wm != nil {
		return fmt.Errorf("workflow manager already registered for ConsignmentService")
	}
	if wm == nil {
		return fmt.Errorf("workflow manager cannot be nil")
	}
	s.wm = wm
	return nil
}

// CompletionHandler is called by the workflow runtime when a workflow completes. It delegates to the appropriate domain-specific handler based on the workflow type.
func (s *Service) CompletionHandler(workflowID string, finalContext map[string]any) error {
	return s.OnWorkflowStatusChanged(context.Background(), s.db, workflowID, model.WorkflowStatusInProgress, model.WorkflowStatusCompleted, nil)
}

// --- WorkflowEventHandler implementation ---

// OnWorkflowStatusChanged handles workflow lifecycle state propagation to consignment domain state.
func (s *Service) OnWorkflowStatusChanged(_ context.Context, tx *gorm.DB, workflowID string, _ model.WorkflowStatus, toStatus model.WorkflowStatus, _ *model.Workflow) error {
	switch toStatus {
	case model.WorkflowStatusCompleted:
		return s.markConsignmentAsFinished(tx, workflowID)
	default:
		return nil
	}
}

// CreateConsignmentShell creates a shell consignment (Stage 1: Trader selects a CHA company).
// The trader's company is resolved from the trader user's OU handle. The specific CHA is not
// assigned yet — that happens at Stage 2 (InitializeConsignmentByID).
func (s *Service) CreateConsignmentShell(ctx context.Context, flow Flow, chaCompanyID string, traderID string) (*DetailDTO, error) {
	chaCompany, err := s.companyService.GetCompanyByID(ctx, chaCompanyID)
	if err != nil {
		return nil, fmt.Errorf("CHA company lookup failed: %w", err)
	}
	if !chaCompany.HasCHA {
		return nil, ErrCompanyNotCHA
	}

	traderUser, err := s.userService.GetUser(traderID)
	if err != nil {
		return nil, fmt.Errorf("trader user lookup failed: %w", err)
	}

	traderCompany, err := s.companyService.GetCompanyByOUHandle(ctx, traderUser.OUHandle)
	if err != nil {
		return nil, fmt.Errorf("trader company lookup failed: %w", err)
	}

	consignment := &Consignment{
		ID:              uuid.NewString(),
		Flow:            flow,
		TraderID:        traderID,
		TraderCompanyID: traderCompany.ID,
		CHACompanyID:    chaCompany.ID,
		State:           Initialized,
		Items:           []Item{},
	}
	if err := s.db.WithContext(ctx).Create(consignment).Error; err != nil {
		return nil, fmt.Errorf("failed to create consignment: %w", err)
	}
	// Reload for response (no workflow nodes at stage 1)
	if err := s.db.WithContext(ctx).First(consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment: %w", err)
	}
	responseDTO, err := s.buildConsignmentDetailDTO(ctx, consignment, nil, make(map[string]hscode.HSCode))
	if err != nil {
		return nil, err
	}
	return responseDTO, nil
}

// InitializeConsignmentByID runs Stage 2: a CHA from the consignment's CHA company picks the
// consignment up, the HS codes are selected, and the workflow is started with the trader
// company data as initial variables.
func (s *Service) InitializeConsignmentByID(
	ctx context.Context,
	consignmentID string,
	hsCodeIDs []string,
	chaID string,
) (*DetailDTO, error) {

	if len(hsCodeIDs) == 0 {
		return nil, fmt.Errorf("at least one HS code ID is required")
	}

	var consignment Consignment
	if err := s.db.WithContext(ctx).First(&consignment, "id = ?", consignmentID).Error; err != nil {
		return nil, fmt.Errorf("consignment not found: %w", err)
	}

	if consignment.State != Initialized {
		return nil, fmt.Errorf("consignment must be in INITIALIZED (current state: %s)", consignment.State)
	}

	// TODO: add support for collapsing multiple HS codes to one workflow.
	// Currently, assumes that there is only one HS code selected.
	if len(hsCodeIDs) > 1 {
		return nil, fmt.Errorf("workflow manager currently supports only one HS code")
	}

	chaRecord, err := s.chaService.GetByID(ctx, chaID)
	if err != nil {
		return nil, fmt.Errorf("CHA lookup failed: %w", err)
	}
	if chaRecord.CompanyID != consignment.CHACompanyID {
		return nil, ErrCHACompanyMismatch
	}

	traderCompany, err := s.companyService.GetCompanyByID(ctx, consignment.TraderCompanyID)
	if err != nil {
		return nil, fmt.Errorf("trader company lookup failed: %w", err)
	}
	traderCompanyVars, err := companyRecordToMap(traderCompany)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trader company: %w", err)
	}
	initialVars := map[string]any{"traderCompany": traderCompanyVars}

	// Prepare items
	items := make([]Item, 0, len(hsCodeIDs))
	for _, hsCodeID := range hsCodeIDs {
		items = append(items, Item{HSCodeID: hsCodeID})
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	consignment.Items = items
	consignment.State = InProgress
	consignment.CHAID = &chaID

	if err := tx.Save(&consignment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update consignment: %w", err)
	}

	var mapping WorkflowTemplateMap
	slog.Error("DO_NOT_SUBMIT Running command:", "hsCodeIDs", hsCodeIDs, "consignment.Flow", consignment.Flow, "query", fmt.Sprintf("hs_code_id = %s AND consignment_flow = %s", hsCodeIDs[0], consignment.Flow))
	err = tx.Model(&WorkflowTemplateMap{}).
		Where("hs_code_id = ? AND consignment_flow = ?", hsCodeIDs[0], consignment.Flow).
		First(&mapping).Error

	if err != nil {
		slog.Error("DO_NOT_SUBMIT Failed to get workflow template", "error", err)
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no workflow template found for HS code %s and flow %s", hsCodeIDs[0], consignment.Flow)
		}
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}
	slog.Error("Found mapping:", "mapping", mapping)

	wt, err := s.templateProvider.GetWorkflowTemplateByIDV2(ctx, mapping.WorkflowTemplateID)
	if err != nil {
		tx.Rollback()
		slog.Error("DO_NOT_SUBMIT Failed to get workflow template", "error", err)
		return nil, fmt.Errorf("failed to get workflow template from provider: %w", err)
	}

	slog.Error("DO_NOT_SUBMIT wt.WorkflowDefinition", "workflowDefinition", wt.WorkflowDefinition)

	if err := s.wm.StartWorkflow(ctx, consignment.ID, wt.WorkflowDefinition, initialVars); err != nil {
		tx.Rollback()
		slog.Error("DO_NOT_SUBMIT Failed to start workflow", "error", err)
		return nil, fmt.Errorf("failed to register workflow: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Reload for response
	if err := s.db.WithContext(ctx).First(&consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment: %w", err)
	}

	var workflowInstance *workflowmanager.WorkflowInstance

	workflowInstance, err = s.wm.GetStatus(ctx, consignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow details: %w", err)
	}

	hsCodeMap, err := s.getHSCodeMap(ctx, consignment.Items)
	if err != nil {
		return nil, err
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, workflowInstance, hsCodeMap)
	if err != nil {
		return nil, err
	}

	return responseDTO, nil
}

// GetConsignmentByID retrieves a consignment by its ID from the database.
func (s *Service) GetConsignmentByID(ctx context.Context, consignmentID string) (*DetailDTO, error) {
	var consignment Consignment
	result := s.db.WithContext(ctx).First(&consignment, "id = ?", consignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve consignment with ID %s: %w", consignmentID, result.Error)
	}

	// Load workflow details (nodes + templates) if workflow exists
	var workflowInstance *workflowmanager.WorkflowInstance
	var err error
	if consignment.State != Initialized {
		workflowInstance, err = s.wm.GetStatus(ctx, consignment.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow details: %w", err)
		}
	}

	hsCodeMap, err := s.getHSCodeMap(ctx, consignment.Items)
	if err != nil {
		return nil, err
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, workflowInstance, hsCodeMap)
	if err != nil {
		return nil, fmt.Errorf("failed to build consignment response DTO: %w", err)
	}

	return responseDTO, nil
}

// ListConsignments returns consignments scoped to a company. For role=trader the caller passes
// TraderCompanyID; for role=cha the caller passes CHACompanyID. Exactly one of the two must be set.
// Scoping is company-based so a user sees all consignments belonging to their company, not only the
// ones they personally created or were individually assigned.
func (s *Service) ListConsignments(ctx context.Context, filter Filter) (*ListResult, error) {
	var baseQuery *gorm.DB
	if filter.CHACompanyID != nil {
		baseQuery = s.db.WithContext(ctx).Model(&Consignment{}).Where("cha_company_id = ?", *filter.CHACompanyID)
	} else if filter.TraderCompanyID != nil {
		baseQuery = s.db.WithContext(ctx).Model(&Consignment{}).Where("trader_company_id = ?", *filter.TraderCompanyID)
	} else {
		return nil, fmt.Errorf("either TraderCompanyID or CHACompanyID must be set in filter")
	}
	return s.listConsignmentsWithBaseQuery(ctx, baseQuery, filter)
}

// listConsignmentsWithBaseQuery runs the shared list logic (filters, count, pagination, DTOs).
func (s *Service) listConsignmentsWithBaseQuery(ctx context.Context, baseQuery *gorm.DB, filter Filter) (*ListResult, error) {
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
		return &ListResult{
			TotalCount: 0,
			Items:      []SummaryDTO{},
			Offset:     finalOffset,
			Limit:      finalLimit,
		}, nil
	}

	var consignments []Consignment
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

	// Batch load HS codes for all JSONB items from all consignments
	var allItems []Item
	for i := range consignments {
		allItems = append(allItems, consignments[i].Items...)
	}
	hsCodeMap, err := s.getHSCodeMap(ctx, allItems)
	if err != nil {
		return nil, err
	}

	// Build Summary DTOs for all consignments
	var consignmentDTOs []SummaryDTO
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
		itemResponseDTOs, err := s.buildConsignmentItemResponseDTOs(c.Items, hsCodeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to load HS code for item in consignment %s: %w", c.ID, err)
		}

		chaID := ""
		if c.CHAID != nil {
			chaID = *c.CHAID
		}

		consignmentDTOs = append(consignmentDTOs, SummaryDTO{
			ID:                         c.ID,
			Flow:                       c.Flow,
			State:                      c.State,
			TraderID:                   c.TraderID,
			TraderCompanyID:            c.TraderCompanyID,
			ChaCompanyID:               c.CHACompanyID,
			ChaID:                      chaID,
			Items:                      itemResponseDTOs,
			CreatedAt:                  c.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                  c.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeCount:          counts.Total,
			CompletedWorkflowNodeCount: counts.Completed,
		})
	}

	return &ListResult{
		TotalCount: totalCount,
		Items:      consignmentDTOs,
		Offset:     finalOffset,
		Limit:      finalLimit,
	}, nil
}

// markConsignmentAsFinished updates the consignment state to FINISHED.
func (s *Service) markConsignmentAsFinished(tx *gorm.DB, consignmentID string) error {
	var consignment Consignment
	if err := tx.First(&consignment, "id = ?", consignmentID).Error; err != nil {
		return fmt.Errorf("failed to retrieve consignment %s: %w", consignmentID, err)
	}
	consignment.State = Finished
	if err := tx.Save(&consignment).Error; err != nil {
		return fmt.Errorf("failed to update consignment %s state to FINISHED: %w", consignmentID, err)
	}
	return nil
}

func (s *Service) getHSCodeMap(ctx context.Context, items []Item) (map[string]hscode.HSCode, error) {
	hsCodeIDs := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, item := range items {
		if !seen[item.HSCodeID] {
			hsCodeIDs = append(hsCodeIDs, item.HSCodeID)
			seen[item.HSCodeID] = true
		}
	}

	if len(hsCodeIDs) == 0 {
		return make(map[string]hscode.HSCode), nil
	}

	hsCodes, err := s.hsCodeService.GetByIDs(ctx, hsCodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch load HS codes: %w", err)
	}

	hsCodeMap := make(map[string]hscode.HSCode, len(hsCodes))
	for _, hsCode := range hsCodes {
		hsCodeMap[hsCode.ID] = hsCode
	}

	return hsCodeMap, nil
}

// buildConsignmentDetailDTO builds a DetailDTO from a Consignment.
// The workflow parameter provides the workflow nodes (nil for INITIALIZED consignments).
func (s *Service) buildConsignmentDetailDTO(
	ctx context.Context,
	consignment *Consignment,
	workflowV2 *workflowmanager.WorkflowInstance,
	hsCodeMap map[string]hscode.HSCode,
) (*DetailDTO, error) {
	itemResponseDTOs, err := s.buildConsignmentItemResponseDTOs(consignment.Items, hsCodeMap)
	if err != nil {
		return nil, err
	}

	nodeResponseDTOs := make([]model.WorkflowNodeResponseDTO, 0)
	edgeResponseDTOs := make([]model.WorkflowEdgeResponseDTO, 0)

	if workflowV2 != nil {
		// Iterate NodeInfo in a deterministic order. Go map iteration is
		// randomized, which surfaces as flaky WorkflowNodes ordering in API
		// responses and tests (e.g. TestConsignmentService_InitializeConsignmentByID_Success).
		// Sorting by node ID is a stable shape; topological order based on
		// Edges would be more user-meaningful and is a follow-up.
		nodeIDs := make([]string, 0, len(workflowV2.NodeInfo))
		for id := range workflowV2.NodeInfo {
			nodeIDs = append(nodeIDs, id)
		}
		sort.Strings(nodeIDs)

		taskTemplateIDs := make([]string, 0, len(workflowV2.NodeInfo))
		for _, id := range nodeIDs {
			node := workflowV2.NodeInfo[id]
			if node.Type == workflowmanager.NodeTypeTask {
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
		for _, id := range nodeIDs {
			node := workflowV2.NodeInfo[id]
			var taskName, taskDescription, taskType string
			var nodeState model.WorkflowNodeState
			if node.Type == workflowmanager.NodeTypeTask {
				if taskTemplate, ok := taskTemplateMap[node.TaskTemplateID]; ok {
					taskName = taskTemplate.Name
					taskDescription = taskTemplate.Description
					taskType = string(taskTemplate.Type)
				} else {
					// Subflow IDs (e.g. FCAU's fcau-pay-app-fee-flow) live in the
					// file-backed taskv2 registry, not workflow_node_templates.
					// Fall back to the template ID until a unified template
					// provider exists.
					taskName = node.TaskTemplateID
					taskType = string(workflowmanager.NodeTypeTask)
				}
			} else {
				taskType = string(node.Type)
			}
			// TODO: clean up translations once the frontend is updated.
			switch node.Status {
			case workflowmanager.NodeStatusRunning:
				nodeState = model.WorkflowNodeStateInProgress
			case workflowmanager.NodeStatusCompleted:
				nodeState = model.WorkflowNodeStateCompleted
			case workflowmanager.NodeStatusFailed:
				nodeState = model.WorkflowNodeStateFailed
			case workflowmanager.NodeStatusNotStarted:
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
				State: nodeState,
			})
		}
		for _, edge := range workflowV2.Edges {
			edgeResponseDTOs = append(edgeResponseDTOs, model.WorkflowEdgeResponseDTO{
				ID:        edge.ID,
				SourceID:  edge.SourceID,
				TargetID:  edge.TargetID,
				Condition: edge.Condition,
			})
		}
	}

	chaID := ""
	if consignment.CHAID != nil {
		chaID = *consignment.CHAID
	}

	return &DetailDTO{
		ID:              consignment.ID,
		Flow:            consignment.Flow,
		State:           consignment.State,
		TraderID:        consignment.TraderID,
		TraderCompanyID: consignment.TraderCompanyID,
		ChaCompanyID:    consignment.CHACompanyID,
		ChaID:           chaID,
		Items:           itemResponseDTOs,
		CreatedAt:       consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes:   nodeResponseDTOs,
		Edges:           edgeResponseDTOs,
	}, nil
}

// companyRecordToMap converts a company.Record to a map[string]any via its JSON tags. The
// marshal/unmarshal round-trip is deliberate: it ties the workflow-variable shape to the same
// JSON contract used elsewhere (REST responses), so a new field on Record with a json tag flows
// through automatically. The reflection cost is negligible here — this runs once per Stage 2
// init, never in a hot path — and the alternative (explicit field map) silently drops new
// fields until someone notices in a workflow step.
func companyRecordToMap(record *company.Record) (map[string]any, error) {
	raw, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// buildConsignmentItemResponseDTOs builds a slice of ItemResponseDTO from ConsignmentItems.
func (s *Service) buildConsignmentItemResponseDTOs(items []Item, hsCodeMap map[string]hscode.HSCode) ([]ItemResponseDTO, error) {
	itemResponseDTOs := make([]ItemResponseDTO, 0, len(items))
	for _, item := range items {
		hsCode, exists := hsCodeMap[item.HSCodeID]
		if !exists {
			return nil, fmt.Errorf("HS code not found for ID %s", item.HSCodeID)
		}
		itemResponseDTOs = append(itemResponseDTOs, ItemResponseDTO{
			HSCode: hscode.ResponseDTO{
				HSCodeID:    hsCode.ID,
				HSCode:      hsCode.HSCode,
				Description: hsCode.Description,
				Category:    hsCode.Category,
			},
		})
	}
	return itemResponseDTOs, nil
}
