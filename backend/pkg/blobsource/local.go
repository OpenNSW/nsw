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

// NewLocal returns a Source backed by a local directory.
//
// If <dir>/manifest.json exists, the source loads blobs according to its
// {"byId": {id: "relpath"}} mapping (mirroring the GitHub backend). Relative
// paths are joined against dir and rejected if they escape dir via "..". This
// lets a local clone of a manifest-based repo be served directly.
//
// Otherwise the source falls back to flat-directory mode: every .json file
// directly in dir is loaded into memory, and the file basename (without
// ".json") is the blob ID. Subdirectories are ignored.
//
// Returns an error if dir is missing or, in flat mode, contains no .json
// files. Payload bytes are not parsed or validated — callers receive raw
// file contents.
func NewLocal(dir string) (Source, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat blobs directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("blobsource: %q is not a directory", dir)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return newLocalManifest(dir, manifestPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat manifest %q: %w", manifestPath, err)
	}

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
