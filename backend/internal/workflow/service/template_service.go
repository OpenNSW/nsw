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
