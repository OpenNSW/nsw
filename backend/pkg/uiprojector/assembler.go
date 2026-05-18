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
	if tp == nil {
		panic("uiprojector: template provider is nil")
	}

	p := make(map[string]Projector, len(projectors))
	for k, v := range projectors {
		p[k] = v
	}

	return &Assembler{
		templateProvider: tp,
		projectors:       p,
	}
}

// Assemble is the "pure" transformation logic.
func (a *Assembler) Assemble(ctx context.Context, blueprint *Blueprint, facts Facts) ([]Section, error) {
	if blueprint == nil {
		return nil, fmt.Errorf("assembler: blueprint is nil")
	}

	sections := make([]Section, 0, len(blueprint.Sections))

	for _, sb := range blueprint.Sections {
		// 1. Visibility Check
		if !ShouldRender(sb, facts) {
			continue
		}

		// 2. Resolve Projector (Fail fast)
		proj, ok := a.projectors[sb.Projector]
		if !ok {
			return nil, fmt.Errorf("assembler: unknown projector %s", sb.Projector)
		}

		// 3. Fetch Template
		templateContent, err := a.templateProvider.GetTemplate(ctx, sb.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("assembler: failed to fetch template %s: %w", sb.TemplateID, err)
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
