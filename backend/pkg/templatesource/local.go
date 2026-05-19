package templatesource

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type localSource struct {
	templates map[string]json.RawMessage
}

// NewLocal reads every .json file directly from dir into memory and returns
// a Source that serves them. The file basename (without ".json") is the
// template ID. Returns an error if dir is missing or any file contains
// invalid JSON.
func NewLocal(dir string) (Source, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory %q: %w", dir, err)
	}

	templates := make(map[string]json.RawMessage)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read template file %q: %w", entry.Name(), err)
		}
		if !json.Valid(data) {
			return nil, fmt.Errorf("template file %q contains invalid JSON", entry.Name())
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		templates[id] = data
		slog.Info("loaded template", "id", id)
	}

	slog.Info("local template source initialized", "dir", dir, "count", len(templates))
	return &localSource{templates: templates}, nil
}

func (s *localSource) GetTemplate(_ context.Context, id string) (json.RawMessage, bool, error) {
	tpl, ok := s.templates[id]
	if !ok {
		return nil, false, nil
	}
	return tpl, true, nil
}

func (s *localSource) Close() error { return nil }
