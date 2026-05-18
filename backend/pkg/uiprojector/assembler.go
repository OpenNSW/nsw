package uiprojector

import (
	"context"
	"fmt"
	"maps"
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

func NewAssembler(tp TemplateProvider, projectors map[string]Projector) (*Assembler, error) {
	if tp == nil {
		return nil, fmt.Errorf("uiprojector: template provider is required")
	}

	return &Assembler{
		templateProvider: tp,
		projectors:       maps.Clone(projectors),
	}, nil
}

// Assemble is the "pure" transformation logic.
func (a *Assembler) Assemble(ctx context.Context, blueprint *Blueprint, facts Facts) (map[string]Section, error) {
	if blueprint == nil {
		return nil, fmt.Errorf("assembler: blueprint is nil")
	}

	// TODO: Should add a cache to cache the frequently fetched templates. Should decide whether the template should be from the TemplateProvider level or This Level.

	sections := make(map[string]Section, len(blueprint.Sections))

	for zone, sb := range blueprint.Sections {
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

		sections[zone] = Section{
			ID:      sb.ID,
			Type:    SectionType(sb.Projector),
			Title:   sb.Title,
			Content: content,
		}
	}

	return sections, nil
}
