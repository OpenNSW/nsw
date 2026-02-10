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

// PreConsignmentService provides operations related to pre-consignments.
type PreConsignmentService struct {
	db                          *gorm.DB
	templateProvider            TemplateProvider
	nodeRepo                    WorkflowNodeRepository
	stateMachine                *WorkflowNodeStateMachine
	preCommitValidationCallback func([]model.WorkflowNode, map[string]any) error
}

// SetPreCommitValidationCallback sets a callback to be executed before transaction commit
// This allows external validation (like task manager registration) to participate in the transaction
func (s *PreConsignmentService) SetPreCommitValidationCallback(callback func([]model.WorkflowNode, map[string]any) error) {
	s.preCommitValidationCallback = callback
}

// NewPreConsignmentService creates a new instance of PreConsignmentService with the provided dependencies.
func NewPreConsignmentService(db *gorm.DB, templateProvider TemplateProvider, nodeRepo WorkflowNodeRepository) *PreConsignmentService {
	return &PreConsignmentService{
		db:               db,
		templateProvider: templateProvider,
		nodeRepo:         nodeRepo,
		stateMachine:     NewWorkflowNodeStateMachine(nodeRepo),
	}
}

// GetTraderPreConsignments retrieves all pre-consignment templates and computes their state
// based on the trader's existing pre-consignments and their dependencies.
func (s *PreConsignmentService) GetTraderPreConsignments(ctx context.Context, traderID string, offset *int, limit *int) (model.TraderPreConsignmentsResponseDTO, error) {
	// Fetch all pre-consignment templates
	var templates []model.PreConsignmentTemplate
	if err := s.db.WithContext(ctx).Find(&templates).Error; err != nil {
		return model.TraderPreConsignmentsResponseDTO{}, fmt.Errorf("failed to retrieve pre-consignment templates: %w", err)
	}

	// Fetch all existing pre-consignments for this trader to determine dependency satisfaction and current states
	var preConsignments []model.PreConsignment
	if err := s.db.WithContext(ctx).
		Where("trader_id = ?", traderID).
		Find(&preConsignments).Error; err != nil {
		return model.TraderPreConsignmentsResponseDTO{}, fmt.Errorf("failed to retrieve completed pre-consignments for trader %s: %w", traderID, err)
	}

	// Build a set of template IDs to PreConsignment for quick lookup
	templateIDToPreConsignment := make(map[uuid.UUID]model.PreConsignment)
	for _, pc := range preConsignments {
		templateIDToPreConsignment[pc.PreConsignmentTemplateID] = pc
	}

	// Build response DTOs with computed state
	allResponseDTOs := make([]model.TraderPreConsignmentResponseDTO, 0, len(templates))
	for _, template := range templates {
		if pc, exists := templateIDToPreConsignment[template.ID]; exists {
			allResponseDTOs = append(allResponseDTOs, model.TraderPreConsignmentResponseDTO{
				ID:             template.ID,
				Name:           template.Name,
				Description:    template.Description,
				DependsOn:      template.DependsOn,
				State:          pc.State,
				PreConsignment: &pc,
			})
			continue
		}

		state := model.PreConsignmentStateReady
		if len(template.DependsOn) > 0 {
			for _, depIDStr := range template.DependsOn {
				depID, err := uuid.Parse(depIDStr)
				if err != nil {
					state = model.PreConsignmentStateLocked
					break
				}
				if depPC, exists := templateIDToPreConsignment[depID]; !exists || depPC.State != model.PreConsignmentStateCompleted {
					state = model.PreConsignmentStateLocked
					break
				}
			}
		}

		dependsOn := template.DependsOn
		if dependsOn == nil {
			dependsOn = []string{}
		}

		allResponseDTOs = append(allResponseDTOs, model.TraderPreConsignmentResponseDTO{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			DependsOn:   dependsOn,
			State:       state,
		})
	}

	// Apply pagination using utility function
	totalCount := int64(len(allResponseDTOs))
	finalOffset, finalLimit := utils.GetPaginationParams(offset, limit)

	start := int64(finalOffset)
	end := start + int64(finalLimit)

	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	paginatedDTOs := allResponseDTOs[start:end]

	return model.TraderPreConsignmentsResponseDTO{
		TotalCount: totalCount,
		Items:      paginatedDTOs,
		Offset:     int64(finalOffset),
		Limit:      int64(finalLimit),
	}, nil
}

// InitializePreConsignment initializes a pre-consignment with its workflow nodes.
// Returns the created pre-consignment response DTO and the new READY workflow nodes.
func (s *PreConsignmentService) InitializePreConsignment(ctx context.Context, createReq *model.CreatePreConsignmentDTO) (*model.PreConsignmentResponseDTO, []model.WorkflowNode, error) {
	if createReq == nil {
		return nil, nil, fmt.Errorf("create request cannot be nil")
	}
	if createReq.TraderID == "" {
		return nil, nil, fmt.Errorf("trader ID cannot be empty")
	}

	return s.initializePreConsignmentInTx(ctx, createReq)
}

// initializePreConsignmentInTx initializes the pre-consignment within a transaction.
func (s *PreConsignmentService) initializePreConsignmentInTx(ctx context.Context, createReq *model.CreatePreConsignmentDTO) (*model.PreConsignmentResponseDTO, []model.WorkflowNode, error) {
	// Get pre-consignment template
	var pcTemplate model.PreConsignmentTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", createReq.PreConsignmentTemplateID).First(&pcTemplate).Error; err != nil {
		return nil, nil, fmt.Errorf("pre-consignment template %s not found: %w", createReq.PreConsignmentTemplateID, err)
	}

	// Validate dependencies are met
	if len(pcTemplate.DependsOn) > 0 {
		var completedCount int64
		if err := s.db.WithContext(ctx).Model(&model.PreConsignment{}).
			Where("trader_id = ? AND pre_consignment_template_id IN ? AND state = ?",
				createReq.TraderID, pcTemplate.DependsOn, model.PreConsignmentStateCompleted).
			Count(&completedCount).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to check dependency completion: %w", err)
		}
		if int(completedCount) < len(pcTemplate.DependsOn) {
			return nil, nil, fmt.Errorf("dependency pre-consignments are not all completed")
		}
	}

	// Fetch the workflow template referenced by the pre-consignment template
	workflowTemplate, err := s.templateProvider.GetWorkflowTemplateByID(ctx, pcTemplate.WorkflowTemplateID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get workflow template %s: %w", pcTemplate.WorkflowTemplateID, err)
	}

	// Fetch node templates
	nodeTemplateIDs := workflowTemplate.GetNodeTemplateIDs()
	nodeTemplates, err := s.templateProvider.GetWorkflowNodeTemplatesByIDs(ctx, nodeTemplateIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve workflow node templates: %w", err)
	}

	// Begin transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create pre-consignment record
	preConsignment := &model.PreConsignment{
		TraderID:                 createReq.TraderID,
		PreConsignmentTemplateID: createReq.PreConsignmentTemplateID,
		State:                    model.PreConsignmentStateInProgress,
		TraderContext:            map[string]any{},
	}
	if err := tx.Create(preConsignment).Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to create pre-consignment: %w", err)
	}

	// Create workflow nodes using the state machine
	_, newReadyWorkflowNodes, err := s.stateMachine.InitializeNodesFromTemplates(
		ctx, tx, ParentRef{PreConsignmentID: &preConsignment.ID}, nodeTemplates,
	)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to create workflow nodes: %w", err)
	}

	// Execute pre-commit validation callback if set (e.g., task manager registration)
	if s.preCommitValidationCallback != nil && len(newReadyWorkflowNodes) > 0 {
		if err := s.preCommitValidationCallback(newReadyWorkflowNodes, preConsignment.TraderContext); err != nil {
			tx.Rollback()
			return nil, nil, fmt.Errorf("pre-commit validation failed: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload pre-consignment with preloaded relationships
	if err := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		Preload("WorkflowNodes.WorkflowNodeTemplate").
		First(preConsignment, "id = ?", preConsignment.ID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to reload pre-consignment with relationships: %w", err)
	}

	// Build response DTO
	responseDTO := s.buildPreConsignmentResponseDTO(preConsignment)

	return responseDTO, newReadyWorkflowNodes, nil
}

// GetPreConsignmentsByTraderID retrieves all pre-consignments for a trader (excluding LOCKED state).
func (s *PreConsignmentService) GetPreConsignmentsByTraderID(ctx context.Context, traderID string) ([]model.PreConsignmentResponseDTO, error) {
	var preConsignments []model.PreConsignment
	result := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		Preload("WorkflowNodes.WorkflowNodeTemplate").
		Where("trader_id = ? AND state != ?", traderID, model.PreConsignmentStateLocked).
		Find(&preConsignments)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve pre-consignments for trader %s: %w", traderID, result.Error)
	}

	if len(preConsignments) == 0 {
		return []model.PreConsignmentResponseDTO{}, nil
	}

	responseDTOs := make([]model.PreConsignmentResponseDTO, 0, len(preConsignments))
	for i := range preConsignments {
		responseDTO := s.buildPreConsignmentResponseDTO(&preConsignments[i])
		responseDTOs = append(responseDTOs, *responseDTO)
	}

	return responseDTOs, nil
}

// GetPreConsignmentByID retrieves a pre-consignment by its ID with loaded workflow nodes and template.
func (s *PreConsignmentService) GetPreConsignmentByID(ctx context.Context, preConsignmentID uuid.UUID) (*model.PreConsignmentResponseDTO, error) {
	var preConsignment model.PreConsignment
	result := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		Preload("WorkflowNodes.WorkflowNodeTemplate").
		First(&preConsignment, "id = ?", preConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve pre-consignment with ID %s: %w", preConsignmentID, result.Error)
	}

	responseDTO := s.buildPreConsignmentResponseDTO(&preConsignment)
	return responseDTO, nil
}

// UpdateWorkflowNodeStateAndPropagateChanges updates the state of a workflow node belonging to a pre-consignment
// and propagates changes to dependent nodes. Returns the new READY nodes and updated trader context.
func (s *PreConsignmentService) UpdateWorkflowNodeStateAndPropagateChanges(ctx context.Context, updateReq *model.UpdateWorkflowNodeDTO) ([]model.WorkflowNode, map[string]any, error) {
	if updateReq == nil {
		return nil, nil, fmt.Errorf("update request cannot be nil")
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	newReadyNodes, traderContext, err := s.updateWorkflowNodeStateAndPropagateChangesInTx(ctx, tx, updateReq)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to update workflow node state: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newReadyNodes, traderContext, nil
}

// updateWorkflowNodeStateAndPropagateChangesInTx handles state transitions within a transaction.
func (s *PreConsignmentService) updateWorkflowNodeStateAndPropagateChangesInTx(ctx context.Context, tx *gorm.DB, updateReq *model.UpdateWorkflowNodeDTO) ([]model.WorkflowNode, map[string]any, error) {
	workflowNode, err := s.nodeRepo.GetWorkflowNodeByIDInTx(ctx, tx, updateReq.WorkflowNodeID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve workflow node with ID %s: %w", updateReq.WorkflowNodeID, err)
	}

	var newReadyNodes []model.WorkflowNode

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

			// Mark pre-consignment as completed if all nodes are done
			if result.AllNodesCompleted {
				if err := s.markPreConsignmentAsCompleted(ctx, tx, *workflowNode.PreConsignmentID); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	// Handle trader context updates
	var traderContext map[string]any
	if len(updateReq.AppendGlobalContext) > 0 {
		traderContext, err = s.appendToPreConsignmentTraderContext(ctx, tx, *workflowNode.PreConsignmentID, updateReq.AppendGlobalContext)
		if err != nil {
			return nil, nil, err
		}
	}

	return newReadyNodes, traderContext, nil
}

// markPreConsignmentAsCompleted updates the pre-consignment state to COMPLETED.
func (s *PreConsignmentService) markPreConsignmentAsCompleted(ctx context.Context, tx *gorm.DB, preConsignmentID uuid.UUID) error {
	var preConsignment model.PreConsignment
	result := tx.WithContext(ctx).First(&preConsignment, "id = ?", preConsignmentID)
	if result.Error != nil {
		return fmt.Errorf("failed to retrieve pre-consignment %s: %w", preConsignmentID, result.Error)
	}

	preConsignment.State = model.PreConsignmentStateCompleted
	if err := tx.WithContext(ctx).Save(&preConsignment).Error; err != nil {
		return fmt.Errorf("failed to update pre-consignment %s state to COMPLETED: %w", preConsignmentID, err)
	}

	return nil
}

// appendToPreConsignmentTraderContext appends key-value pairs to the pre-consignment's trader context.
func (s *PreConsignmentService) appendToPreConsignmentTraderContext(ctx context.Context, tx *gorm.DB, preConsignmentID uuid.UUID, appendContext map[string]any) (map[string]any, error) {
	var preConsignment model.PreConsignment
	result := tx.WithContext(ctx).First(&preConsignment, "id = ?", preConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve pre-consignment %s: %w", preConsignmentID, result.Error)
	}

	if preConsignment.TraderContext == nil {
		preConsignment.TraderContext = make(map[string]any)
	}
	maps.Copy(preConsignment.TraderContext, appendContext)

	if err := tx.WithContext(ctx).Save(&preConsignment).Error; err != nil {
		return nil, fmt.Errorf("failed to update pre-consignment %s trader context: %w", preConsignmentID, err)
	}

	return preConsignment.TraderContext, nil
}

// buildPreConsignmentResponseDTO builds a PreConsignmentResponseDTO from a PreConsignment with preloaded relationships.
func (s *PreConsignmentService) buildPreConsignmentResponseDTO(preConsignment *model.PreConsignment) *model.PreConsignmentResponseDTO {
	// Build WorkflowNodeResponseDTOs from preloaded nodes
	nodeResponseDTOs := make([]model.WorkflowNodeResponseDTO, 0, len(preConsignment.WorkflowNodes))
	for _, node := range preConsignment.WorkflowNodes {
		nodeResponseDTOs = append(nodeResponseDTOs, model.WorkflowNodeResponseDTO{
			ID:        node.ID,
			CreatedAt: node.CreatedAt.Format(time.RFC3339),
			UpdatedAt: node.UpdatedAt.Format(time.RFC3339),
			WorkflowNodeTemplate: model.WorkflowNodeTemplateResponseDTO{
				Name:        node.WorkflowNodeTemplate.Name,
				Description: node.WorkflowNodeTemplate.Description,
				Type:        string(node.WorkflowNodeTemplate.Type),
			},
			State:     node.State,
			DependsOn: node.DependsOn,
		})
	}

	dependsOn := preConsignment.PreConsignmentTemplate.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}

	return &model.PreConsignmentResponseDTO{
		ID:            preConsignment.ID,
		TraderID:      preConsignment.TraderID,
		State:         preConsignment.State,
		TraderContext: preConsignment.TraderContext,
		CreatedAt:     preConsignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     preConsignment.UpdatedAt.Format(time.RFC3339),
		PreConsignmentTemplate: model.PreConsignmentTemplateResponseDTO{
			ID:          preConsignment.PreConsignmentTemplate.ID,
			Name:        preConsignment.PreConsignmentTemplate.Name,
			Description: preConsignment.PreConsignmentTemplate.Description,
			DependsOn:   dependsOn,
		},
		WorkflowNodes: nodeResponseDTOs,
	}
}
