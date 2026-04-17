package template

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type TemplateRegistry struct {
	templates         map[string]TemplateDefinition
	handlers          map[TemplateType]TemplateHandler
	defaultTemplateID string
}

func NewTemplateRegistry(dir string, defaultTemplateID string) (*TemplateRegistry, error) {
	registry := &TemplateRegistry{
		templates:         make(map[string]TemplateDefinition),
		handlers:          make(map[TemplateType]TemplateHandler),
		defaultTemplateID: defaultTemplateID,
	}

	if err := registry.RegisterHandler(NewFormHandler()); err != nil {
		return nil, err
	}

	if err := registry.loadTemplates(dir); err != nil {
		return nil, err
	}

	if defaultTemplateID != "" {
		if _, ok := registry.templates[defaultTemplateID]; !ok {
			return nil, fmt.Errorf("default template %q not found in %q", defaultTemplateID, dir)
		}
	}

	slog.Info("template registry initialized", "count", len(registry.templates), "defaultTemplateID", defaultTemplateID)
	return registry, nil
}

func (r *TemplateRegistry) RegisterHandler(h TemplateHandler) error {
	if h == nil {
		return fmt.Errorf("template handler is required")
	}

	handlerType := h.Type()
	if handlerType == "" {
		return fmt.Errorf("template handler type is required")
	}

	r.handlers[handlerType] = h
	return nil
}

func (r *TemplateRegistry) GetTemplate(id string) (TemplateDefinition, error) {
	templateDef, ok := r.templates[id]
	if !ok {
		return TemplateDefinition{}, fmt.Errorf("template %q not found", id)
	}

	return templateDef, nil
}

func (r *TemplateRegistry) GetDefaultTemplate() (TemplateDefinition, error) {
	if r.defaultTemplateID == "" {
		return TemplateDefinition{}, fmt.Errorf("no default template configured")
	}

	return r.GetTemplate(r.defaultTemplateID)
}

func (r *TemplateRegistry) loadTemplates(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read templates directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read template file %q: %w", entry.Name(), err)
		}

		if !json.Valid(data) {
			return fmt.Errorf("template file %q contains invalid JSON", entry.Name())
		}

		var templateDef TemplateDefinition
		if err := json.Unmarshal(data, &templateDef); err != nil {
			return fmt.Errorf("failed to parse template file %q: %w", entry.Name(), err)
		}

		templateID := strings.TrimSuffix(entry.Name(), ".json")
		if templateDef.ID != "" && templateDef.ID != templateID {
			return fmt.Errorf("template ID %q in file %q does not match filename", templateDef.ID, entry.Name())
		}
		templateDef.ID = templateID

		if err := r.validateTemplate(templateID, templateDef); err != nil {
			return err
		}

		r.templates[templateID] = templateDef
		slog.Debug("loaded template", "id", templateID)
	}

	return nil
}

func (r *TemplateRegistry) validateTemplate(templateID string, templateDef TemplateDefinition) error {
	if err := r.validateSection(templateID, "view", templateDef.View); err != nil {
		return err
	}

	if err := r.validateSection(templateID, "action", templateDef.Action); err != nil {
		return err
	}

	return nil
}

func (r *TemplateRegistry) validateSection(templateID string, sectionName string, section TemplateSection) error {
	if section.Type == "" {
		return fmt.Errorf("template %q section %q is missing type", templateID, sectionName)
	}

	h, ok := r.handlers[section.Type]
	if !ok {
		return fmt.Errorf("template %q section %q has unsupported type %q", templateID, sectionName, section.Type)
	}

	if len(section.Content) == 0 {
		return fmt.Errorf("template %q section %q is missing content", templateID, sectionName)
	}

	var content map[string]any
	if err := json.Unmarshal(section.Content, &content); err != nil {
		return fmt.Errorf("template %q section %q content must be an object: %w", templateID, sectionName, err)
	}

	if err := h.Validate(content); err != nil {
		return fmt.Errorf("template %q section %q failed validation: %w", templateID, sectionName, err)
	}

	return nil
}
