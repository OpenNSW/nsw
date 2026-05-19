package service

import (
	"context"

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

// GetWorkflowTemplateByIDV2 retrieves a workflow template by its ID.
func (s *TemplateService) GetWorkflowTemplateByIDV2(ctx context.Context, id string) (*model.WorkflowTemplateV2, error) {
	var template model.WorkflowTemplateV2
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
