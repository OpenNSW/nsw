package service

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// FileBasedTemplateService implements TemplateProviderV2 by reading workflow
// definitions from an in-memory TaskTemplateRegistry populated from JSON files.
type FileBasedTemplateService struct {
	registry *orchestrator.TaskTemplateRegistry
}

// NewFileBasedTemplateService creates a FileBasedTemplateService backed by registry.
func NewFileBasedTemplateService(registry *orchestrator.TaskTemplateRegistry) *FileBasedTemplateService {
	return &FileBasedTemplateService{registry: registry}
}

// GetWorkflowTemplateByID looks up a workflow definition by ID and returns it
// as a WorkflowTemplateV2. Returns an error if no definition with that ID exists.
func (s *FileBasedTemplateService) GetWorkflowTemplateByID(_ context.Context, id string) (*model.WorkflowTemplateV2, error) {
	def, ok := s.registry.GetWorkflow(id)
	if !ok {
		return nil, fmt.Errorf("workflow template %q not found", id)
	}
	return &model.WorkflowTemplateV2{
		BaseModel:          model.BaseModel{ID: def.ID},
		Name:               def.Name,
		Version:            fmt.Sprintf("%d", def.Version),
		WorkflowDefinition: def,
	}, nil
}
