package internal

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	onetrade "github.com/OpenNSW/one-trade-templates"
)

// FormsSubdir is the subdirectory under the config root where form files live.
const FormsSubdir = "forms"

// FormStore holds loaded form definitions ({ schema, uiSchema }) keyed by form ID.
// The form ID is the filename (without the .json extension).
type FormStore struct {
	forms map[string]json.RawMessage
}

// NewFormStore reads all .json files from <configDir>/forms into memory.
// When useOneTrade is true, forms from the embedded one-trade-templates FS are
// also loaded; their IDs are the path relative to templates/ without the .json
// suffix (e.g. "npqs/1-application/userinput_jsonform").
func NewFormStore(configDir string, useOneTrade bool) (*FormStore, error) {
	dir := filepath.Join(configDir, FormsSubdir)
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
		if !json.Valid(data) {
			return nil, fmt.Errorf("form file %q contains invalid JSON", entry.Name())
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		forms[id] = data
		slog.Info("loaded form", "id", id)
	}

	if useOneTrade {
		sub, err := fs.Sub(onetrade.FS, "templates")
		if err != nil {
			return nil, fmt.Errorf("failed to create onetrade sub-FS: %w", err)
		}
		if err := loadOneTradeForms(forms, sub); err != nil {
			return nil, err
		}
	}

	slog.Info("form store initialized", "count", len(forms))
	return &FormStore{forms: forms}, nil
}

// loadOneTradeForms walks fsys for *_jsonform.json files and adds them to forms.
func loadOneTradeForms(forms map[string]json.RawMessage, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_jsonform.json") {
			return err
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read onetrade form %q: %w", path, err)
		}
		if !json.Valid(data) {
			return fmt.Errorf("onetrade form %q contains invalid JSON", path)
		}

		id := strings.TrimSuffix(path, ".json")
		forms[id] = data
		slog.Info("loaded onetrade form", "id", id)
		return nil
	})
}

// GetForm returns the raw JSON for the given form ID and whether it was found.
func (store *FormStore) GetForm(id string) (json.RawMessage, bool) {
	form, ok := store.forms[id]
	return form, ok
}
