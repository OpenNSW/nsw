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

// Load loads templates into registry from source.
// When source is empty or "embedded", templates are fetched from the
// one-trade-templates GitHub repository at the default main-branch URL.
// Any other value is treated as a local filesystem directory path.
func Load(registry *orchestrator.TaskTemplateRegistry, source string) error {
	if source == "" || source == "embedded" {
		client := newOneTradeClient(defaultOneTradeBaseURL)
		return loadFromRemote(registry, client)
	}
	return LoadFromDir(registry, source)
}

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
		registerJSON(registry, path, data)
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("templates: walk %s: %w", templatesDir, walkErr)
	}
	return nil
}

// loadFromRemote fetches the manifest from client and registers each template
// listed in byId with the registry.
func loadFromRemote(registry *orchestrator.TaskTemplateRegistry, client *oneTradeClient) error {
	if registry == nil {
		return fmt.Errorf("templates: registry is required")
	}

	manifest, err := client.fetchManifest()
	if err != nil {
		return fmt.Errorf("templates: %w", err)
	}

	for _, path := range manifest.ByID {
		data, err := client.fetchFile(path)
		if err != nil {
			return fmt.Errorf("templates: %w", err)
		}
		registerJSON(registry, path, data)
	}

	return nil
}

// registerJSON classifies a single JSON file and registers it with the registry.
func registerJSON(registry *orchestrator.TaskTemplateRegistry, path string, data []byte) {
	// 1. Task template entry — has template_id + plugin_name
	var entry orchestrator.TaskTemplateEntry
	if err := json.Unmarshal(data, &entry); err == nil && entry.ID != "" && entry.PluginName != "" {
		registry.Register(entry)
		// slog.Info("templates: registered task template",
		// 	"templateId", entry.ID, "taskType", entry.TaskType, "plugin", entry.PluginName)
		return
	}

	// 2. Sub-workflow definition — has id + nodes
	var workflowDef engine.WorkflowDefinition
	if err := json.Unmarshal(data, &workflowDef); err == nil && workflowDef.ID != "" && len(workflowDef.Nodes) > 0 {
		registry.RegisterWorkflow(workflowDef)
		// slog.Info("templates: registered sub-workflow",
		// 	"id", workflowDef.ID, "name", workflowDef.Name, "nodes", len(workflowDef.Nodes))
		return
	}

	// 3. Generic template — has top-level "id" (e.g. JSONForms schema)
	var generic struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &generic); err == nil && generic.ID != "" {
		registry.RegisterGenericTemplate(generic.ID, data)
		// slog.Info("templates: registered generic template", "id", generic.ID, "path", path)
		return
	}

	slog.Warn("templates: unrecognised JSON, skipped", "path", path)
}
