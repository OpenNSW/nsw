package uiprojector

import (
	"context"
	"fmt"
)

// TemplateProvider abstracts the resolution of TemplateID to raw bytes.
type TemplateProvider interface {
	GetTemplate(ctx context.Context, templateID string) ([]byte, error)
}

// Assembler transforms a Blueprint and Facts into a list of rendered Sections.
type Assembler struct {
	templateProvider TemplateProvider
	projectors       map[string]Projector
}

func NewAssembler(tp TemplateProvider, projectors map[string]Projector) *Assembler {
	return &Assembler{
		templateProvider: tp,
		projectors:       projectors,
	}
}

// Assemble is the "pure" transformation logic.
func (a *Assembler) Assemble(ctx context.Context, blueprint *Blueprint, facts Facts) ([]Section, error) {
	var sections []Section

	evaluator := NewVisibilityEvaluator()

	for _, sb := range blueprint.Sections {
		// 1. Visibility Check
		if !evaluator.ShouldRender(sb, facts) {
			continue
		}

		// 2. Fetch Template
		templateContent, err := a.templateProvider.GetTemplate(ctx, sb.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("assembler: failed to fetch template %s: %w", sb.TemplateID, err)
		}

		// 3. Resolve Projector
		proj, ok := a.projectors[sb.Projector]
		if !ok {
			return nil, fmt.Errorf("assembler: unknown projector %s", sb.Projector)
		}

		// 4. Pluck Data from Registry via DataKey
		var sectionData any
		if sb.DataKey != "" {
			sectionData = facts.Data[sb.DataKey]
		} else {
			sectionData = facts.Data
		}

		// 5. Project
		content, err := proj.Project(ctx, templateContent, sectionData)
		if err != nil {
			return nil, fmt.Errorf("assembler: projection failed for section %s: %w", sb.ID, err)
		}

		sections = append(sections, Section{
			ID:      sb.ID,
			Type:    SectionType(sb.Projector),
			Title:   sb.Title,
			Content: content,
		})
	}

	return sections, nil
}
