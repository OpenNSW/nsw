package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FormsSubdir is the subdirectory under the config root where form files live.
const FormsSubdir = "forms"

// FormStore holds loaded form definitions ({ schema, uiSchema }) keyed by form ID.
// The form ID is the filename (without the .json extension).
type FormStore struct {
	forms map[string]json.RawMessage
}

// NewFormStore reads all .json files from <configDir>/forms into memory.
// When oneTradeBaseURL is non-empty, forms are also fetched from the one-trade-templates
// GitHub repository at that base URL (e.g. the main branch raw URL or a pinned SHA/tag).
// An empty oneTradeBaseURL disables remote loading.
func NewFormStore(configDir string, oneTradeBaseURL string) (*FormStore, error) {
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
		// slog.Info("loaded form", "id", id)
	}

	if oneTradeBaseURL != "" {
		client := newOneTradeClient(oneTradeBaseURL)
		if err := loadOneTradeForms(forms, client); err != nil {
			return nil, err
		}
	}

	slog.Info("form store initialized", "count", len(forms))
	return &FormStore{forms: forms}, nil
}

// loadOneTradeForms fetches the manifest from the remote client and loads all
// *_jsonform.json entries into forms, keyed by the manifest's byId key.
func loadOneTradeForms(forms map[string]json.RawMessage, client *oneTradeClient) error {
	manifest, err := client.fetchManifest()
	if err != nil {
		return fmt.Errorf("failed to fetch onetrade manifest: %w", err)
	}

	for id, path := range manifest.ByID {
		if !strings.HasSuffix(path, "_jsonform.json") {
			continue
		}

		data, err := client.fetchFile(path)
		if err != nil {
			return fmt.Errorf("failed to fetch onetrade form %q: %w", path, err)
		}
		if !json.Valid(data) {
			return fmt.Errorf("onetrade form %q contains invalid JSON", path)
		}

		forms[id] = data
		// slog.Info("loaded onetrade form", "id", id)
	}

	return nil
}

// GetForm returns the raw JSON for the given form ID and whether it was found.
func (store *FormStore) GetForm(id string) (json.RawMessage, bool) {
	form, ok := store.forms[id]
	return form, ok
}
