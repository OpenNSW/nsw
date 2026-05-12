// Package template loads task and workflow templates from a directory of JSON
// files into an nsw-task-flow TaskTemplateRegistry.
package template

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	engine "github.com/OpenNSW/go-temporal-workflow"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
)

// LoadFromDir walks templatesDir recursively and registers each JSON file as
// either a TaskTemplateEntry, a sub-WorkflowDefinition, or a generic template
// (e.g. JSONForms schema). Files that don't match any pattern are skipped.
func LoadFromDir(registry *orchestrator.TaskTemplateRegistry, templatesDir string) error {
	if registry == nil {
		return fmt.Errorf("templates: registry is required")
	}
	walkErr := filepath.WalkDir(templatesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// 1. Task template entry — has template_id + plugin_name
		var entry orchestrator.TaskTemplateEntry
		if err := json.Unmarshal(data, &entry); err == nil && entry.TemplateID != "" && entry.PluginName != "" {
			registry.Register(entry)
			slog.Info("templates: registered task template",
				"templateId", entry.TemplateID, "taskType", entry.TaskType, "plugin", entry.PluginName)
			return nil
		}

		// 2. Sub-workflow definition — has id + nodes
		var workflowDef engine.WorkflowDefinition
		if err := json.Unmarshal(data, &workflowDef); err == nil && workflowDef.ID != "" && len(workflowDef.Nodes) > 0 {
			registry.RegisterWorkflow(workflowDef)
			slog.Info("templates: registered sub-workflow",
				"id", workflowDef.ID, "name", workflowDef.Name, "nodes", len(workflowDef.Nodes))
			return nil
		}

		// 3. Generic template — has top-level "id" (e.g. JSONForms schema)
		var generic struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &generic); err == nil && generic.ID != "" {
			registry.RegisterGenericTemplate(generic.ID, data)
			slog.Info("templates: registered generic template", "id", generic.ID, "path", path)
			return nil
		}

		slog.Warn("templates: unrecognised JSON, skipped", "path", path)
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("templates: walk %s: %w", templatesDir, walkErr)
	}
	return nil
}
