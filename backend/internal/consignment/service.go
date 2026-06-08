package consignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	workflowmanager "github.com/OpenNSW/core/workflow"
	tfstore "github.com/OpenNSW/core/taskflow/store"

	"github.com/OpenNSW/nsw/backend/internal/hscode"
	"github.com/OpenNSW/nsw/backend/internal/profile/cha"
	"github.com/OpenNSW/nsw/backend/internal/profile/company"
	"github.com/OpenNSW/nsw/backend/internal/profile/user"
	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
	"github.com/OpenNSW/nsw/backend/internal/workflow/service"
	"github.com/OpenNSW/nsw/backend/pkg/pagination"
)

// TaskStore is the narrow interface needed from taskv2 package to load task records.
type TaskStore interface {
	GetAllTasks(ctx context.Context, parentWorkflowID string) []tfstore.TaskRecord
}

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
	taskStore        TaskStore
}

// NewService creates a new instance of Service.
func NewService(
	db *gorm.DB,
	templateProvider service.TemplateProvider,
	chaService cha.Service,
	companyService company.Service,
	userService user.Service,
	hsCodeService *hscode.Service,
	taskStore TaskStore,
) *Service {
	return &Service{
		db:               db,
		templateProvider: templateProvider,
		chaService:       chaService,
		companyService:   companyService,
		userService:      userService,
		hsCodeService:    hsCodeService,
		taskStore:        taskStore,
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
		CHACompanyID:    &chaCompany.ID,
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
	responseDTO, err := s.buildConsignmentDetailDTO(ctx, consignment, make(map[string]hscode.HSCode))
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
	if consignment.CHACompanyID == nil || chaRecord.CompanyID != *consignment.CHACompanyID {
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
	err = tx.Model(&WorkflowTemplateMap{}).
		Where("hs_code_id = ? AND consignment_flow = ?", hsCodeIDs[0], consignment.Flow).
		First(&mapping).Error

	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no workflow template found for HS code %s and flow %s", hsCodeIDs[0], consignment.Flow)
		}
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}

	wt, err := s.templateProvider.GetWorkflowTemplateByIDV2(ctx, mapping.WorkflowTemplateID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to get workflow template from provider: %w", err)
	}

	if err := s.wm.StartWorkflow(ctx, consignment.ID, wt.WorkflowDefinition, initialVars); err != nil {
		tx.Rollback()
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

	if _, err := s.wm.GetStatus(ctx, consignment.ID); err != nil {
		return nil, fmt.Errorf("failed to get workflow details: %w", err)
	}

	hsCodeMap, err := s.getHSCodeMap(ctx, consignment.Items)
	if err != nil {
		return nil, err
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, hsCodeMap)
	if err != nil {
		return nil, err
	}

	return responseDTO, nil
}

// directStartExportWorkflowTemplateID is the top-level workflow started immediately by
// CreateAndStartConsignment. CHA and HS code selection happen inside this workflow's own
// tasks (trade_1_cha_selection, trade_2_hscode_selection) as workflow variables
// (trade.cha_id, trade.hs_codes) rather than as an upfront trader/CHA handoff.
const directStartExportWorkflowTemplateID = "trade-export-v1"

// CreateAndStartConsignment creates an export consignment and starts its workflow directly,
// in one step — replacing the two-stage trader-creates-shell → CHA-claims-with-HS-code handoff
// for flows whose entire CHA/HS-code selection now happens inside the workflow itself.
func (s *Service) CreateAndStartConsignment(ctx context.Context, traderID string) (*DetailDTO, error) {
	traderUser, err := s.userService.GetUser(traderID)
	if err != nil {
		return nil, fmt.Errorf("trader user lookup failed: %w", err)
	}

	traderCompany, err := s.companyService.GetCompanyByOUHandle(ctx, traderUser.OUHandle)
	if err != nil {
		return nil, fmt.Errorf("trader company lookup failed: %w", err)
	}

	traderCompanyVars, err := companyRecordToMap(traderCompany)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trader company: %w", err)
	}
	initialVars := map[string]any{"traderCompany": traderCompanyVars}

	wt, err := s.templateProvider.GetWorkflowTemplateByIDV2(ctx, directStartExportWorkflowTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}

	consignment := &Consignment{
		ID:              uuid.NewString(),
		Flow:            FlowExport,
		TraderID:        traderID,
		TraderCompanyID: traderCompany.ID,
		State:           InProgress,
		Items:           []Item{},
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(consignment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create consignment: %w", err)
	}

	if err := s.wm.StartWorkflow(ctx, consignment.ID, wt.WorkflowDefinition, initialVars); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to register workflow: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	if err := s.db.WithContext(ctx).First(consignment, "id = ?", consignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload consignment: %w", err)
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, consignment, make(map[string]hscode.HSCode))
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

	// Confirm the workflow is reachable if one exists; node details now come
	// from task records rather than this status snapshot.
	if consignment.State != Initialized {
		if _, err := s.wm.GetStatus(ctx, consignment.ID); err != nil {
			return nil, fmt.Errorf("failed to get workflow details: %w", err)
		}
	}

	hsCodeMap, err := s.getHSCodeMap(ctx, consignment.Items)
	if err != nil {
		return nil, err
	}

	responseDTO, err := s.buildConsignmentDetailDTO(ctx, &consignment, hsCodeMap)
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
	finalOffset, finalLimit := pagination.ResolvePaginationParams(filter.Offset, filter.Limit)

	// Each call returns a fresh GORM chain so LIMIT/OFFSET/ORDER on the list
	// query cannot leak into the count query — consistent with hscode and
	// profile/company services.
	filteredQuery := func() *gorm.DB {
		q := baseQuery
		if filter.State != nil {
			q = q.Where("state = ?", *filter.State)
		}
		if filter.Flow != nil {
			q = q.Where("flow = ?", *filter.Flow)
		}
		return q
	}

	var consignments []Consignment
	// NOTE: We do NOT preload WorkflowNodes here to improve performance
	if err := filteredQuery().
		Offset(finalOffset).
		Limit(finalLimit).
		Order("created_at DESC").
		Find(&consignments).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve consignments: %w", err)
	}

	var totalCount int64
	if len(consignments) < finalLimit && finalOffset == 0 {
		totalCount = int64(len(consignments))
	} else {
		if err := filteredQuery().Count(&totalCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count filtered consignments: %w", err)
		}
	}

	if len(consignments) == 0 {
		result := pagination.NewPageResult([]SummaryDTO{}, totalCount, finalOffset, finalLimit)
		return &result, nil
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
		chaCompanyID := ""
		if c.CHACompanyID != nil {
			chaCompanyID = *c.CHACompanyID
		}

		consignmentDTOs = append(consignmentDTOs, SummaryDTO{
			ID:                         c.ID,
			Flow:                       c.Flow,
			State:                      c.State,
			TraderID:                   c.TraderID,
			TraderCompanyID:            c.TraderCompanyID,
			ChaCompanyID:               chaCompanyID,
			ChaID:                      chaID,
			Items:                      itemResponseDTOs,
			CreatedAt:                  c.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                  c.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeCount:          counts.Total,
			CompletedWorkflowNodeCount: counts.Completed,
		})
	}

	result := pagination.NewPageResult(consignmentDTOs, totalCount, finalOffset, finalLimit)
	return &result, nil
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
func (s *Service) buildConsignmentDetailDTO(
	ctx context.Context,
	consignment *Consignment,
	hsCodeMap map[string]hscode.HSCode,
) (*DetailDTO, error) {
	itemResponseDTOs, err := s.buildConsignmentItemResponseDTOs(consignment.Items, hsCodeMap)
	if err != nil {
		return nil, err
	}

	nodeResponseDTOs, err := s.buildNodeDTOsFromTaskRecords(ctx, consignment.ID)
	if err != nil {
		return nil, err
	}

	chaID := ""
	if consignment.CHAID != nil {
		chaID = *consignment.CHAID
	}
	chaCompanyID := ""
	if consignment.CHACompanyID != nil {
		chaCompanyID = *consignment.CHACompanyID
	}

	return &DetailDTO{
		ID:              consignment.ID,
		Flow:            consignment.Flow,
		State:           consignment.State,
		TraderID:        consignment.TraderID,
		TraderCompanyID: consignment.TraderCompanyID,
		ChaCompanyID:    chaCompanyID,
		ChaID:           chaID,
		Items:           itemResponseDTOs,
		CreatedAt:       consignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       consignment.UpdatedAt.Format(time.RFC3339),
		WorkflowNodes:   nodeResponseDTOs,
	}, nil
}

// buildNodeDTOsFromTaskRecords queries tasks via the TaskStore by root_workflow_id and converts each
// non-SYSTEM record into a WorkflowNodeResponseDTO for the consignment detail response.
// This replaces the direct database access: every task record—including those from child
// workflows spawned by SPLIT_TASK—shares the same root_workflow_id (the consignment ID),
// so a single exact-match query captures the complete task picture.
func (s *Service) buildNodeDTOsFromTaskRecords(ctx context.Context, consignmentID string) ([]model.WorkflowNodeResponseDTO, error) {
	if s.taskStore == nil {
		return nil, fmt.Errorf("task store not initialized")
	}
	tasks := s.taskStore.GetAllTasks(ctx, consignmentID)

	dtos := make([]model.WorkflowNodeResponseDTO, 0, len(tasks))
	for _, t := range tasks {
		if t.TaskType == "SYSTEM" {
			continue
		}
		var nodeState model.WorkflowNodeState
		switch t.State {
		case "COMPLETED":
			nodeState = model.WorkflowNodeStateCompleted
		case "FAILED":
			nodeState = model.WorkflowNodeStateFailed
		default:
			nodeState = model.WorkflowNodeStateInProgress
		}
		dtos = append(dtos, model.WorkflowNodeResponseDTO{
			ID:        t.TaskID,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeTemplate: model.WorkflowNodeTemplateResponseDTO{
				Name: taskDisplayName(t.ActiveTaskTemplateID, t.RenderConfig),
				Type: t.TaskType,
			},
			State: nodeState,
		})
	}
	return dtos, nil
}

// taskDisplayName extracts the human-readable title from a task's render config workspace
// section, falling back to the active template ID when no title is present.
func taskDisplayName(templateID string, renderConfig json.RawMessage) string {
	if len(renderConfig) > 0 {
		var rc struct {
			Sections map[string]struct {
				Title string `json:"title"`
			} `json:"sections"`
		}
		if err := json.Unmarshal(renderConfig, &rc); err == nil {
			if ws, ok := rc.Sections["workspace"]; ok && ws.Title != "" {
				return ws.Title
			}
		}
	}
	return templateID
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
