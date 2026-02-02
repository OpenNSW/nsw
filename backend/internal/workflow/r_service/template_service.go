package r_service

import (
	"context"

	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
func (s *TemplateService) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID string, flow string) (*r_model.WorkflowTemplate, error) {
	var workflowTemplate r_model.WorkflowTemplate
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

// GetWorkflowNodeTemplatesByIDs retrieves workflow node templates by their IDs.
func (s *TemplateService) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]r_model.WorkflowNodeTemplate, error) {
	var templates []r_model.WorkflowNodeTemplate
	result := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&templates)
	if result.Error != nil {
		return nil, result.Error
	}
	return templates, nil
}
