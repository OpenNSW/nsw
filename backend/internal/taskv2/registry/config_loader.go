package registry

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	engine "github.com/OpenNSW/go-temporal-workflow"
	"github.com/OpenNSW/nsw-task-flow/orchestrator"
)

// LoadConfigsInto walks rootDir using a folder-as-task convention. Each
// immediate subfolder of rootDir represents one Task; within it, the loader
// classifies files by name:
//
//	workflow.json or *_workflow.json → engine.WorkflowDefinition  (RegisterWorkflow)
//	render.json                      → uiprojector blueprint + Task metadata (RegisterGeneric)
//	*_jsonform.json                  → JSONForms schema           (RegisterGeneric)
//	anything else (userinput.json,
//	  reviewerinput.json,
//	  payment.json, …)               → orchestrator.SubTaskTemplate (RegisterSubTask)
//
// After processing a task folder the loader synthesizes an
// orchestrator.TaskTemplate from the discovered workflow.json + render.json:
//
//	TaskTemplate{
//	    ID:             <workflow.json id>,
//	    Type:           <render.json type>,
//	    WorkflowID:     <workflow.json id>,
//	    RenderConfigID: <render.json id>,
//	}
//
// Each task folder MUST contain exactly one workflow file and one render.json,
// otherwise LoadConfigsInto returns an error.
func LoadConfigsInto(reg *InMemRegistry, rootDir string) error {
	if reg == nil {
		return fmt.Errorf("config loader: registry is nil")
	}

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return fmt.Errorf("config loader: read %s: %w", rootDir, err)
	}

	taskFolders := 0
	for _, e := range entries {
		if e.IsDir() {
			if err := loadTaskFolder(reg, filepath.Join(rootDir, e.Name())); err != nil {
				return err
			}
			taskFolders++
		} else {
			// Load the top level workflow definition
			name := e.Name()
			if name == "workflow.json" || strings.HasSuffix(name, "_workflow.json") {
				path := filepath.Join(rootDir, name)
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("read %s: %w", path, err)
				}
				if !json.Valid(data) {
					return fmt.Errorf("invalid JSON in %s", path)
				}
				var w engine.WorkflowDefinition
				if err := json.Unmarshal(data, &w); err != nil {
					return fmt.Errorf("workflow %s: %w", path, err)
				}
				if w.ID == "" {
					return fmt.Errorf("workflow %s: missing id", path)
				}
				reg.RegisterWorkflow(w)
				slog.Info("registered top-level workflow", "id", w.ID, "path", path)
			}
		}
	}

	if taskFolders == 0 {
		return fmt.Errorf("config loader: no task subfolders found under %s", rootDir)
	}
	slog.Info("config loader done", "root", rootDir, "task_folders", taskFolders)
	return nil
}

// loadTaskFolder processes a single task folder. It classifies every file
// within the folder (recursively, in case of nested subtask groupings),
// registers each piece in reg, then synthesizes and registers the
// TaskTemplate for the folder as a whole.
func loadTaskFolder(reg *InMemRegistry, dir string) error {
	var workflowID string
	var renderID string
	var taskType string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if !json.Valid(data) {
			return fmt.Errorf("invalid JSON in %s", path)
		}

		switch {
		case name == "workflow.json" || strings.HasSuffix(name, "_workflow.json"):
			var w engine.WorkflowDefinition
			if err := json.Unmarshal(data, &w); err != nil {
				return fmt.Errorf("workflow %s: %w", path, err)
			}
			if w.ID == "" {
				return fmt.Errorf("workflow %s: missing id", path)
			}
			reg.RegisterWorkflow(w)
			if workflowID != "" && workflowID != w.ID {
				return fmt.Errorf("task folder %s: multiple workflow ids (%s and %s)", dir, workflowID, w.ID)
			}
			workflowID = w.ID
			slog.Info("registered workflow", "id", w.ID, "path", path)

		case name == "render.json":
			var probe struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &probe); err != nil {
				return fmt.Errorf("render %s: %w", path, err)
			}
			if probe.ID == "" {
				return fmt.Errorf("render %s: missing id", path)
			}
			reg.RegisterGeneric(probe.ID, data)
			if renderID != "" && renderID != probe.ID {
				return fmt.Errorf("task folder %s: multiple render ids (%s and %s)", dir, renderID, probe.ID)
			}
			renderID = probe.ID
			taskType = probe.Type
			slog.Info("registered render config", "id", probe.ID, "type", probe.Type, "path", path)

		case strings.HasSuffix(name, "_jsonform.json"):
			id, err := extractID(data)
			if err != nil {
				return fmt.Errorf("jsonform %s: %w", path, err)
			}
			reg.RegisterGeneric(id, data)
			slog.Info("registered jsonform", "id", id, "path", path)

		default:
			var st orchestrator.SubTaskTemplate
			if err := json.Unmarshal(data, &st); err != nil {
				return fmt.Errorf("subtask %s: %w", path, err)
			}
			if st.ID == "" {
				return fmt.Errorf("subtask %s: missing id", path)
			}
			reg.RegisterSubTask(st)
			slog.Info("registered subtask", "id", st.ID, "path", path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("config loader: walk %s: %w", dir, err)
	}

	if workflowID == "" {
		return fmt.Errorf("task folder %s: workflow.json missing or has no id", dir)
	}
	if renderID == "" {
		return fmt.Errorf("task folder %s: render.json missing or has no id", dir)
	}

	reg.RegisterTask(orchestrator.TaskTemplate{
		ID:             workflowID,
		Type:           taskType,
		WorkflowID:     workflowID,
		RenderConfigID: renderID,
	})
	slog.Info("registered task", "id", workflowID, "type", taskType, "render_config_id", renderID, "folder", dir)
	return nil
}

// extractID peeks at the top-level "id" string without a full unmarshal.
func extractID(data []byte) (string, error) {
	var probe struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return "", err
	}
	if probe.ID == "" {
		return "", fmt.Errorf("missing id field")
	}
	return probe.ID, nil
}
