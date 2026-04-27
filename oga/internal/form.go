package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FormStore holds loaded form definitions keyed by form ID.
type FormStore struct {
	forms         map[string]json.RawMessage
	defaultFormID string
}

// NewFormStore reads all .json files from dir into memory.
// The form ID is the filename without the .json extension.
func NewFormStore(dir string, defaultFormID string) (*FormStore, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read forms directory %q: %w", dir, err)
	}

	forms := make(map[string]json.RawMessage)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read form file %q: %w", entry.Name(), err)
		}

		// Validate that it's valid JSON
		if !json.Valid(data) {
			return nil, fmt.Errorf("form file %q contains invalid JSON", entry.Name())
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		forms[id] = data
		slog.Info("loaded form", "id", id)
	}

	if defaultFormID != "" {
		if _, ok := forms[defaultFormID]; !ok {
			return nil, fmt.Errorf("default form %q not found in %q", defaultFormID, dir)
		}
	}

	slog.Info("form store initialized", "count", len(forms), "defaultFormID", defaultFormID)
	return &FormStore{forms: forms, defaultFormID: defaultFormID}, nil
}

// GetForm returns the raw JSON for the given form ID, or an error if not found.
func (fs *FormStore) GetForm(id string) (json.RawMessage, error) {
	form, ok := fs.forms[id]
	if !ok {
		return nil, fmt.Errorf("form %q not found", id)
	}
	return form, nil
}

// GetDefaultForm returns the default form, or an error if none is configured.
func (fs *FormStore) GetDefaultForm() (json.RawMessage, error) {
	if fs.defaultFormID == "" {
		return nil, fmt.Errorf("no default form configured")
	}
	return fs.GetForm(fs.defaultFormID)
}

// FormIDFromTaskCode returns the form ID associated with the given task code.
func FormIDFromTaskCode(taskCode string) string {
	// TODO: this is a temporary implementation. The form ID construction logic may evolve as we support more complex scenarios.
	// For example, we might want to include the task code, or have a mapping of task codes to form IDs in the config.
	return taskCode
}
