package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/OpenNSW/nsw/internal/auth"
	workflowmanager "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/utils"
)

// PreConsignmentService provides operations related to pre-consignments.
// It also implements WorkflowEventHandler for domain-specific lifecycle callbacks.
type PreConsignmentService struct {
	db               *gorm.DB
	templateProvider TemplateProvider
	workflowManager  workflowmanager.Manager
}

// NewPreConsignmentService creates a new instance of PreConsignmentService with the provided dependencies.
func NewPreConsignmentService(db *gorm.DB, templateProvider TemplateProvider, workflowManager workflowmanager.Manager) *PreConsignmentService {
	return &PreConsignmentService{
		db:               db,
		templateProvider: templateProvider,
		workflowManager:  workflowManager,
	}
}

// --- WorkflowEventHandler implementation ---

// OnWorkflowStatusChanged handles workflow lifecycle state propagation to pre-consignment domain state.
func (s *PreConsignmentService) OnWorkflowStatusChanged(_ context.Context, tx *gorm.DB, workflowID uuid.UUID, _ model.WorkflowStatus, toStatus model.WorkflowStatus, workflow *model.Workflow) error {
	var preConsignment model.PreConsignment
	if err := tx.First(&preConsignment, "id = ?", workflowID).Error; err != nil {
		return fmt.Errorf("failed to retrieve pre-consignment %s: %w", workflowID, err)
	}

	switch toStatus {
	case model.WorkflowStatusCompleted:
		preConsignment.State = model.PreConsignmentStateCompleted
		if err := tx.Save(&preConsignment).Error; err != nil {
			return fmt.Errorf("failed to update pre-consignment %s state to COMPLETED: %w", workflowID, err)
		}
		if workflow == nil {
			return fmt.Errorf("workflow payload cannot be nil for completed state")
		}
		if err := s.syncTraderContextToAuth(tx, &preConsignment, workflow.GlobalContext); err != nil {
			return fmt.Errorf("failed to sync trader context to auth: %w", err)
		}
	}

	return nil
}

// GetTraderPreConsignments retrieves a paginated list of pre-consignment templates and computes their state
// based on the trader's existing pre-consignments and their dependencies.
func (s *PreConsignmentService) GetTraderPreConsignments(ctx context.Context, traderID string, offset *int, limit *int) (model.TraderPreConsignmentsResponseDTO, error) {
	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(offset, limit)

	// Get total count of templates first for pagination
	var totalCount int64
	if err := s.db.WithContext(ctx).Model(&model.PreConsignmentTemplate{}).Count(&totalCount).Error; err != nil {
		return model.TraderPreConsignmentsResponseDTO{}, fmt.Errorf("failed to count pre-consignment templates: %w", err)
	}

	if totalCount == 0 {
		return model.TraderPreConsignmentsResponseDTO{
			TotalCount: 0,
			Items:      []model.TraderPreConsignmentResponseDTO{},
			Offset:     int64(finalOffset),
			Limit:      int64(finalLimit),
		}, nil
	}

	// Fetch pre-consignment templates for the current page
	var templates []model.PreConsignmentTemplate
	if err := s.db.WithContext(ctx).
		Order("name ASC").
		Offset(finalOffset).
		Limit(finalLimit).
		Find(&templates).Error; err != nil {
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

	// Build response DTOs with computed state ONLY for the fetched templates (the current page)
	responseDTOs := make([]model.TraderPreConsignmentResponseDTO, 0, len(templates))
	for _, template := range templates {
		if pc, exists := templateIDToPreConsignment[template.ID]; exists {
			responseDTOs = append(responseDTOs, model.TraderPreConsignmentResponseDTO{
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

		responseDTOs = append(responseDTOs, model.TraderPreConsignmentResponseDTO{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			DependsOn:   dependsOn,
			State:       state,
		})
	}

	return model.TraderPreConsignmentsResponseDTO{
		TotalCount: totalCount,
		Items:      responseDTOs,
		Offset:     int64(finalOffset),
		Limit:      int64(finalLimit),
	}, nil
}

// InitializePreConsignment initializes a pre-consignment with its workflow.
// Returns the created pre-consignment response DTO.
func (s *PreConsignmentService) InitializePreConsignment(
	ctx context.Context,
	createReq *model.CreatePreConsignmentDTO,
	traderId string,
	initialTraderContext map[string]any,
) (*model.PreConsignmentResponseDTO, error) {
	if createReq == nil {
		return nil, fmt.Errorf("create request cannot be nil")
	}
	if traderId == "" {
		return nil, fmt.Errorf("trader ID cannot be empty")
	}
	if initialTraderContext == nil {
		initialTraderContext = make(map[string]any)
	}

	return s.initializePreConsignmentInTx(ctx, createReq, traderId, initialTraderContext)
}

// initializePreConsignmentInTx initializes the pre-consignment within a transaction.
func (s *PreConsignmentService) initializePreConsignmentInTx(
	ctx context.Context,
	createReq *model.CreatePreConsignmentDTO,
	traderId string,
	initialTraderContext map[string]any,
) (*model.PreConsignmentResponseDTO, error) {
	// Get pre-consignment template
	var pcTemplate model.PreConsignmentTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", createReq.PreConsignmentTemplateID).First(&pcTemplate).Error; err != nil {
		return nil, fmt.Errorf("pre-consignment template %s not found: %w", createReq.PreConsignmentTemplateID, err)
	}

	// Validate dependencies are met
	if len(pcTemplate.DependsOn) > 0 {
		var completedCount int64
		if err := s.db.WithContext(ctx).Model(&model.PreConsignment{}).
			Where("trader_id = ? AND pre_consignment_template_id IN ? AND state = ?",
				traderId, pcTemplate.DependsOn, model.PreConsignmentStateCompleted).
			Count(&completedCount).Error; err != nil {
			return nil, fmt.Errorf("failed to check dependency completion: %w", err)
		}
		if int(completedCount) < len(pcTemplate.DependsOn) {
			return nil, fmt.Errorf("dependency pre-consignments are not all completed")
		}
	}

	// Fetch the workflow template referenced by the pre-consignment template
	workflowTemplate, err := s.templateProvider.GetWorkflowTemplateByID(ctx, pcTemplate.WorkflowTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template %s: %w", pcTemplate.WorkflowTemplateID, err)
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
		TraderID:                 traderId,
		PreConsignmentTemplateID: createReq.PreConsignmentTemplateID,
		State:                    model.PreConsignmentStateInProgress,
	}
	if err := tx.Create(preConsignment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create pre-consignment: %w", err)
	}

	// Register workflow with the manager (creates Workflow entity + nodes + registers with TM)
	if err := s.workflowManager.StartWorkflowInstance(ctx, tx, preConsignment.ID, []model.WorkflowTemplate{*workflowTemplate}, nil, initialTraderContext, s); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to register workflow: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload pre-consignment with template for response
	if err := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		First(preConsignment, "id = ?", preConsignment.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload pre-consignment: %w", err)
	}

	// Get workflow details for response
	wf, err := s.workflowManager.GetWorkflowInstance(ctx, preConsignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow details: %w", err)
	}

	responseDTO := s.buildPreConsignmentResponseDTO(preConsignment, wf)
	return responseDTO, nil
}

// GetPreConsignmentsByTraderID retrieves all pre-consignments for a trader (excluding LOCKED state).
func (s *PreConsignmentService) GetPreConsignmentsByTraderID(ctx context.Context, traderID string) ([]model.PreConsignmentResponseDTO, error) {
	var preConsignments []model.PreConsignment
	result := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
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
		// Get workflow details for each pre-consignment
		wf, err := s.workflowManager.GetWorkflowInstance(ctx, preConsignments[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow details for pre-consignment %s: %w", preConsignments[i].ID, err)
		}
		responseDTO := s.buildPreConsignmentResponseDTO(&preConsignments[i], wf)
		responseDTOs = append(responseDTOs, *responseDTO)
	}

	return responseDTOs, nil
}

// GetPreConsignmentByID retrieves a pre-consignment by its ID with loaded workflow nodes and template.
func (s *PreConsignmentService) GetPreConsignmentByID(ctx context.Context, preConsignmentID uuid.UUID) (*model.PreConsignmentResponseDTO, error) {
	var preConsignment model.PreConsignment
	result := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		First(&preConsignment, "id = ?", preConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve pre-consignment with ID %s: %w", preConsignmentID, result.Error)
	}

	wf, err := s.workflowManager.GetWorkflowInstance(ctx, preConsignment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow details: %w", err)
	}

	responseDTO := s.buildPreConsignmentResponseDTO(&preConsignment, wf)
	return responseDTO, nil
}

// syncTraderContextToAuth synchronizes the trader context (from the workflow's global context) to the auth system.
// This is called when a pre-consignment is completed to persist accumulated context.
func (s *PreConsignmentService) syncTraderContextToAuth(tx *gorm.DB, preConsignment *model.PreConsignment, traderContext map[string]any) error {
	var uc auth.UserContext
	result := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", preConsignment.TraderID).
		First(&uc)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			contextJSON, err := json.Marshal(traderContext)
			if err != nil {
				return fmt.Errorf("failed to marshal user context: %w", err)
			}

			uc = auth.UserContext{
				UserID:      preConsignment.TraderID,
				UserContext: contextJSON,
			}

			if err := tx.Create(&uc).Error; err != nil {
				return fmt.Errorf("failed to create user context: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to query user context: %w", result.Error)
	}

	var existingContext map[string]any
	if len(uc.UserContext) > 0 {
		if err := json.Unmarshal(uc.UserContext, &existingContext); err != nil {
			return fmt.Errorf("failed to unmarshal existing user context: %w", err)
		}
	} else {
		existingContext = make(map[string]any)
	}

	for k, v := range traderContext {
		existingContext[k] = v
	}

	updatedJSON, err := json.Marshal(existingContext)
	if err != nil {
		return fmt.Errorf("failed to marshal updated user context: %w", err)
	}

	uc.UserContext = updatedJSON
	if err := tx.Save(&uc).Error; err != nil {
		return fmt.Errorf("failed to update user context: %w", err)
	}

	return nil
}

// buildPreConsignmentResponseDTO builds a PreConsignmentResponseDTO from a PreConsignment.
// The workflow parameter provides the workflow nodes and global context (trader context).
func (s *PreConsignmentService) buildPreConsignmentResponseDTO(preConsignment *model.PreConsignment, workflow *model.Workflow) *model.PreConsignmentResponseDTO {
	var nodeResponseDTOs []model.WorkflowNodeResponseDTO
	if workflow != nil {
		nodeResponseDTOs = make([]model.WorkflowNodeResponseDTO, 0, len(workflow.WorkflowNodes))
		for _, node := range workflow.WorkflowNodes {
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
	}
	if nodeResponseDTOs == nil {
		nodeResponseDTOs = []model.WorkflowNodeResponseDTO{}
	}

	// Populate TraderContext from the Workflow's GlobalContext for backward compatibility
	var traderContext map[string]any
	if workflow != nil {
		traderContext = workflow.GlobalContext
	}

	dependsOn := preConsignment.PreConsignmentTemplate.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}

	return &model.PreConsignmentResponseDTO{
		ID:            preConsignment.ID,
		TraderID:      preConsignment.TraderID,
		State:         preConsignment.State,
		TraderContext: traderContext,
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
