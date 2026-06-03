package blobsource

// Local-manifest mode lets NewLocal serve a directory that ships a
// manifest.json with a {"byId": {id: "relpath"}} mapping (the same shape used
// by the GitHub backend). This is useful when a local clone of a
// manifest-based repo keeps its blobs nested under subdirectories rather than
// flat at the directory root. NewLocal in local.go dispatches here when a
// manifest.json is present.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func newLocalManifest(dir, manifestPath string) (Source, error) {
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %q: %w", manifestPath, err)
	}
	var m manifestData
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest %q: %w", manifestPath, err)
	}
	if m.ByID == nil {
		return nil, fmt.Errorf("manifest %q has no byId field", manifestPath)
	}

	absDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %q: %w", dir, err)
	}

	blobs := make(map[string][]byte, len(m.ByID))
	for id, rel := range m.ByID {
		if rel == "" {
			return nil, fmt.Errorf("manifest %q: empty path for id %q", manifestPath, id)
		}
		fullPath, err := safeJoin(absDir, rel)
		if err != nil {
			return nil, fmt.Errorf("manifest %q: %w", manifestPath, err)
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read blob file %q for id %q: %w", fullPath, id, err)
		}
		blobs[id] = data
		slog.Info("loaded blob", "id", id, "path", rel)
	}

	slog.Info("local blob source initialized (manifest mode)",
		"dir", dir, "manifestEntries", len(blobs))
	return &localSource{blobs: blobs}, nil
}

// safeJoin joins rel onto absDir and rejects results that escape absDir via
// ".." or absolute paths. It also resolves symlinks so that a symlink pointing
// outside the directory cannot bypass the check. absDir must already be the
// real path (resolved via filepath.EvalSymlinks by the caller).
func safeJoin(absDir, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("manifest path %q must be relative", rel)
	}
	cleaned := filepath.Clean(filepath.Join(absDir, rel))
	// filepath.Rel is more robust than string-prefix checks across OSes.
	relPath, err := filepath.Rel(absDir, cleaned)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("manifest path %q escapes manifest directory", rel)
	}
	// Resolve symlinks to prevent traversal via symlinks pointing outside absDir.
	real, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", fmt.Errorf("manifest path %q: %w", rel, err)
	}
	realRel, err := filepath.Rel(absDir, real)
	if err != nil || strings.HasPrefix(realRel, "..") {
		return "", fmt.Errorf("manifest path %q escapes manifest directory via symlink", rel)
	}
	return real, nil
}
