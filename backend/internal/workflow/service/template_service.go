package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
)

type TemplateService struct {
	db       *gorm.DB
	registry WorkflowDefinitionProvider
}

// NewTemplateService creates a new instance of TemplateService.
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{
		db: db,
	}
}

// WithRegistry associates an in-memory WorkflowDefinitionProvider with this service.
func (s *TemplateService) WithRegistry(registry WorkflowDefinitionProvider) *TemplateService {
	s.registry = registry
	return s
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
	if s.registry == nil {
		return nil, fmt.Errorf("template service: workflow registry is not configured")
	}
	def, ok := s.registry.GetWorkflow(id)
	if !ok {
		return nil, fmt.Errorf("template service: workflow %q not found in registry", id)
	}
	return &model.WorkflowTemplateV2{
		Name:               id,
		Version:            "1",
		WorkflowDefinition: def,
	}, nil
}
