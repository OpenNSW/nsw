package service

import (
	"context"

	"github.com/google/uuid"
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
func (s *TemplateService) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
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

// GetWorkflowTemplateByID retrieves a workflow template by its ID.
func (s *TemplateService) GetWorkflowTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowTemplate, error) {
	var workflowTemplate model.WorkflowTemplate
	result := s.db.WithContext(ctx).First(&workflowTemplate, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &workflowTemplate, nil
}

// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
func (s *TemplateService) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]model.WorkflowNodeTemplate, error) {
	var templates []model.WorkflowNodeTemplate
	result := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&templates)
	if result.Error != nil {
		return nil, result.Error
	}
	return templates, nil
}

// GetWorkflowNodeTemplateByID retrieves a workflow node template by its ID.
func (s *TemplateService) GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error) {
	var template model.WorkflowNodeTemplate
	result := s.db.WithContext(ctx).First(&template, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &template, nil
}
