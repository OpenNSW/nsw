package blobsource

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type localSource struct {
	blobs map[string][]byte
}

// NewLocal reads every .json file directly from dir into memory and returns
// a Source that serves them. The file basename (without ".json") is the
// blob ID. Returns an error if dir is missing or contains no .json files.
//
// Note: discovery is restricted to .json files for now. Payload bytes are
// not parsed or validated — callers receive raw file contents.
func NewLocal(dir string) (Source, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read blobs directory %q: %w", dir, err)
	}

	blobs := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read blob file %q: %w", entry.Name(), err)
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		blobs[id] = data
		slog.Info("loaded blob", "id", id)
	}

	if len(blobs) == 0 {
		return nil, fmt.Errorf("blobsource: no .json files found in %q", dir)
	}

	slog.Info("local blob source initialized", "dir", dir, "count", len(blobs))
	return &localSource{blobs: blobs}, nil
}

func (s *localSource) Get(_ context.Context, id string) ([]byte, bool, error) {
	blob, ok := s.blobs[id]
	if !ok {
		return nil, false, nil
	}
	return blob, true, nil
}

func (s *localSource) Close() error { return nil }
