package preconsignment

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	workflowmanager "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/OpenNSW/nsw/utils"
)

// Service provides operations related to pre-consignments.
// It also implements WorkflowEventHandler for domain-specific lifecycle callbacks.
type Service struct {
	db               *gorm.DB
	templateProvider service.TemplateProvider
	workflowManager  workflowmanager.Manager
}

// NewService creates a new instance of Service with the provided dependencies.
func NewService(db *gorm.DB, templateProvider service.TemplateProvider, workflowManager workflowmanager.Manager) *Service {
	return &Service{
		db:               db,
		templateProvider: templateProvider,
		workflowManager:  workflowManager,
	}
}

// --- WorkflowEventHandler implementation ---

// OnWorkflowStatusChanged handles workflow lifecycle state propagation to pre-consignment domain state.
func (s *Service) OnWorkflowStatusChanged(_ context.Context, tx *gorm.DB, workflowID string, _ model.WorkflowStatus, toStatus model.WorkflowStatus, workflow *model.Workflow) error {
	var pc PreConsignment
	if err := tx.First(&pc, "id = ?", workflowID).Error; err != nil {
		return fmt.Errorf("failed to retrieve pre-consignment %s: %w", workflowID, err)
	}

	switch toStatus {
	case model.WorkflowStatusCompleted:
		pc.State = StateCompleted
		if err := tx.Save(&pc).Error; err != nil {
			return fmt.Errorf("failed to update pre-consignment %s state to COMPLETED: %w", workflowID, err)
		}
		if workflow == nil {
			return fmt.Errorf("workflow payload cannot be nil for completed state")
		}
		if err := s.syncTraderContextToAuth(tx, &pc, workflow.GlobalContext); err != nil {
			return fmt.Errorf("failed to sync trader context to auth: %w", err)
		}
	}

	return nil
}

// GetTraderPreConsignments retrieves a paginated list of pre-consignment templates and computes their state
// based on the trader's existing pre-consignments and their dependencies.
func (s *Service) GetTraderPreConsignments(ctx context.Context, traderID string, offset *int, limit *int) (TraderPreConsignmentsResponseDTO, error) {
	// Apply pagination with defaults and limits
	finalOffset, finalLimit := utils.GetPaginationParams(offset, limit)

	// Get total count of templates first for pagination
	var totalCount int64
	if err := s.db.WithContext(ctx).Model(&Template{}).Count(&totalCount).Error; err != nil {
		return TraderPreConsignmentsResponseDTO{}, fmt.Errorf("failed to count pre-consignment templates: %w", err)
	}

	if totalCount == 0 {
		return TraderPreConsignmentsResponseDTO{
			TotalCount: 0,
			Items:      []TraderPreConsignmentResponseDTO{},
			Offset:     int64(finalOffset),
			Limit:      int64(finalLimit),
		}, nil
	}

	// Fetch pre-consignment templates for the current page
	var templates []Template
	if err := s.db.WithContext(ctx).
		Order("name ASC").
		Offset(finalOffset).
		Limit(finalLimit).
		Find(&templates).Error; err != nil {
		return TraderPreConsignmentsResponseDTO{}, fmt.Errorf("failed to retrieve pre-consignment templates: %w", err)
	}

	// Fetch all existing pre-consignments for this trader to determine dependency satisfaction and current states
	var preConsignments []PreConsignment
	if err := s.db.WithContext(ctx).
		Where("trader_id = ?", traderID).
		Find(&preConsignments).Error; err != nil {
		return TraderPreConsignmentsResponseDTO{}, fmt.Errorf("failed to retrieve completed pre-consignments for trader %s: %w", traderID, err)
	}

	// Build a set of template IDs to PreConsignment for quick lookup
	templateIDToPreConsignment := make(map[string]PreConsignment)
	for _, pc := range preConsignments {
		templateIDToPreConsignment[pc.PreConsignmentTemplateID] = pc
	}

	// Build response DTOs with computed state ONLY for the fetched templates (the current page)
	responseDTOs := make([]TraderPreConsignmentResponseDTO, 0, len(templates))
	for _, template := range templates {
		if pc, exists := templateIDToPreConsignment[template.ID]; exists {
			responseDTOs = append(responseDTOs, TraderPreConsignmentResponseDTO{
				ID:             template.ID,
				Name:           template.Name,
				Description:    template.Description,
				DependsOn:      template.DependsOn,
				State:          pc.State,
				PreConsignment: &pc,
			})
			continue
		}

		state := StateReady
		if len(template.DependsOn) > 0 {
			for _, depIDStr := range template.DependsOn {
				if depPC, exists := templateIDToPreConsignment[depIDStr]; !exists || depPC.State != StateCompleted {
					state = StateLocked
					break
				}
			}
		}

		dependsOn := template.DependsOn
		if dependsOn == nil {
			dependsOn = []string{}
		}

		responseDTOs = append(responseDTOs, TraderPreConsignmentResponseDTO{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			DependsOn:   dependsOn,
			State:       state,
		})
	}

	return TraderPreConsignmentsResponseDTO{
		TotalCount: totalCount,
		Items:      responseDTOs,
		Offset:     int64(finalOffset),
		Limit:      int64(finalLimit),
	}, nil
}

// InitializePreConsignment initializes a pre-consignment with its workflow.
// Returns the created pre-consignment response DTO.
func (s *Service) InitializePreConsignment(
	ctx context.Context,
	createReq *CreatePreConsignmentDTO,
	traderId string,
	initialTraderContext map[string]any,
) (*ResponseDTO, error) {
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
func (s *Service) initializePreConsignmentInTx(
	ctx context.Context,
	createReq *CreatePreConsignmentDTO,
	traderId string,
	initialTraderContext map[string]any,
) (*ResponseDTO, error) {
	// Get pre-consignment template
	var pcTemplate Template
	if err := s.db.WithContext(ctx).Where("id = ?", createReq.PreConsignmentTemplateID).First(&pcTemplate).Error; err != nil {
		return nil, fmt.Errorf("pre-consignment template %s not found: %w", createReq.PreConsignmentTemplateID, err)
	}

	// Validate dependencies are met
	if len(pcTemplate.DependsOn) > 0 {
		var completedCount int64
		if err := s.db.WithContext(ctx).Model(&PreConsignment{}).
			Where("trader_id = ? AND pre_consignment_template_id IN ? AND state = ?",
				traderId, pcTemplate.DependsOn, StateCompleted).
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

	pc := &PreConsignment{
		TraderID:                 traderId,
		PreConsignmentTemplateID: createReq.PreConsignmentTemplateID,
		State:                    StateInProgress,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pc).Error; err != nil {
			return fmt.Errorf("failed to create pre-consignment: %w", err)
		}
		if err := s.workflowManager.StartWorkflowInstance(ctx, tx, pc.ID, []model.WorkflowTemplate{*workflowTemplate}, initialTraderContext, s); err != nil {
			return fmt.Errorf("failed to register workflow: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Reload pre-consignment with template for response
	if err := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		First(pc, "id = ?", pc.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload pre-consignment: %w", err)
	}

	// Get workflow details for response
	wf, err := s.workflowManager.GetWorkflowInstance(ctx, pc.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow details: %w", err)
	}

	responseDTO := s.buildPreConsignmentResponseDTO(pc, wf)
	return responseDTO, nil
}

// GetPreConsignmentsByTraderID retrieves all pre-consignments for a trader (excluding LOCKED state).
func (s *Service) GetPreConsignmentsByTraderID(ctx context.Context, traderID string) ([]ResponseDTO, error) {
	var preConsignments []PreConsignment
	result := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		Preload("Workflow").
		Preload("Workflow.WorkflowNodes").
		Preload("Workflow.WorkflowNodes.WorkflowNodeTemplate").
		Where("trader_id = ? AND state != ?", traderID, StateLocked).
		Find(&preConsignments)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve pre-consignments for trader %s: %w", traderID, result.Error)
	}

	if len(preConsignments) == 0 {
		return []ResponseDTO{}, nil
	}

	responseDTOs := make([]ResponseDTO, 0, len(preConsignments))
	for i := range preConsignments {
		responseDTO := s.buildPreConsignmentResponseDTO(&preConsignments[i], preConsignments[i].Workflow)
		responseDTOs = append(responseDTOs, *responseDTO)
	}

	return responseDTOs, nil
}

// GetPreConsignmentByID retrieves a pre-consignment by its ID with loaded workflow nodes and template.
func (s *Service) GetPreConsignmentByID(ctx context.Context, preConsignmentID string) (*ResponseDTO, error) {
	var pc PreConsignment
	result := s.db.WithContext(ctx).
		Preload("PreConsignmentTemplate").
		First(&pc, "id = ?", preConsignmentID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve pre-consignment with ID %s: %w", preConsignmentID, result.Error)
	}

	wf, err := s.workflowManager.GetWorkflowInstance(ctx, pc.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow details: %w", err)
	}

	responseDTO := s.buildPreConsignmentResponseDTO(&pc, wf)
	return responseDTO, nil
}

// syncTraderContextToAuth synchronizes the trader context (from the workflow's global context) to the user profile.
// This is called when a pre-consignment is completed to persist accumulated context.
// It updates the Data field on the company profile Record.
// TODO: Once the Company Profile Service is implemented, we should merge the pre-consignment results with the existing company profile data instead of overwriting it.
// TODO: This function name and signature may need to be refactored as well once we have a clearer picture of the data flow and ownership between pre-consignment, company profile, and user profile.
func (s *Service) syncTraderContextToAuth(_ *gorm.DB, _ *PreConsignment, _ map[string]any) error {
	return nil
}

// buildPreConsignmentResponseDTO builds a ResponseDTO from a PreConsignment.
// The workflow parameter provides the workflow nodes and global context (trader context).
func (s *Service) buildPreConsignmentResponseDTO(pc *PreConsignment, workflow *model.Workflow) *ResponseDTO {
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

	dependsOn := pc.PreConsignmentTemplate.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}

	return &ResponseDTO{
		ID:            pc.ID,
		TraderID:      pc.TraderID,
		State:         pc.State,
		TraderContext: traderContext,
		CreatedAt:     pc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     pc.UpdatedAt.Format(time.RFC3339),
		PreConsignmentTemplate: TemplateResponseDTO{
			ID:          pc.PreConsignmentTemplate.ID,
			Name:        pc.PreConsignmentTemplate.Name,
			Description: pc.PreConsignmentTemplate.Description,
			DependsOn:   dependsOn,
		},
		WorkflowNodes: nodeResponseDTOs,
	}
}
