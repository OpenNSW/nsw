package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService creates a new instance of TemplateService.
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{
		db: db,
	}
}

// GetWorkflowTemplateByHSCodeIDAndFlow retrieves the workflow template associated with a given HS code and consignment flow.
func (s *TemplateService) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	var workflowTemplate model.WorkflowTemplate
	result := s.db.WithContext(ctx).Table("workflow_templates").
		Select("workflow_templates.*").
		Joins("JOIN workflow_template_maps ON workflow_templates.id = workflow_template_maps.workflow_template_id").
		Where("workflow_template_maps.hs_code_id = ? AND workflow_template_maps.consignment_flow = ?", hsCodeID, flow).
		First(&workflowTemplate)
	if result.Error != nil {
		return nil, result.Error
	}

	return &workflowTemplate, nil
}

func (s *TemplateService) GetWorkflowTemplateByHSCodeIDAndFlowV2(
	ctx context.Context,
	hsCodeID string,
	flow model.ConsignmentFlow,
) (*model.WorkflowTemplateV2, error) {

	var workflowTemplate model.WorkflowTemplateV2

	result := s.db.WithContext(ctx).
		Model(&model.WorkflowTemplateV2{}).
		Joins("JOIN workflow_template_maps_v2 ON workflow_template_v2.id = workflow_template_maps_v2.workflow_template_id").
		Where(
			"workflow_template_maps_v2.hs_code_id = ? AND workflow_template_maps_v2.consignment_flow = ?",
			hsCodeID,
			flow,
		).
		Order("workflow_template_v2.version DESC"). // optional but future-proof
		First(&workflowTemplate)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &workflowTemplate, nil
}

// GetWorkflowTemplateByID retrieves a workflow template by its ID.
func (s *TemplateService) GetWorkflowTemplateByID(ctx context.Context, id string) (*model.WorkflowTemplate, error) {
	var workflowTemplate model.WorkflowTemplate
	result := s.db.WithContext(ctx).First(&workflowTemplate, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &workflowTemplate, nil
}

// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
func (s *TemplateService) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []string) ([]model.WorkflowNodeTemplate, error) {
	var templates []model.WorkflowNodeTemplate
	result := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&templates)
	if result.Error != nil {
		return nil, result.Error
	}
	return templates, nil
}

// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
func (s *TemplateService) GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error) {
	var template model.WorkflowNodeTemplate
	result := s.db.WithContext(ctx).First(&template, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &template, nil
}

// GetEndNodeTemplate retrieves the special end node template.
// Assumes there is only one end node template in the system, identified by its type.
func (s *TemplateService) GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error) {
	var template model.WorkflowNodeTemplate
	result := s.db.WithContext(ctx).Where("type = ?", model.WorkFlowNodeTypeEndNode).First(&template)
	if result.Error != nil {
		return nil, result.Error
	}
	return &template, nil
}
