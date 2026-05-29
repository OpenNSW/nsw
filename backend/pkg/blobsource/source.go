// Package blobsource resolves opaque byte blobs by ID. It exposes a Source
// interface with two implementations: a local-folder reader and a
// GitHub-manifest-backed loader. The package treats payloads as opaque bytes
// and never inspects their structure — interpretation is the caller's job.
package blobsource

import (
	"context"
	"fmt"
)

// Source resolves blobs by ID.
//
// Return contract:
//   - (bytes, true, nil)   — blob found and returned.
//   - (nil,   false, nil)  — the ID is not known to this source. Callers should
//     treat this as "skip and continue without the blob".
//   - (nil,   false, err)  — fetch failed for an otherwise-known ID. Callers
//     should warn-log and continue without the blob.
type Source interface {
	Get(ctx context.Context, id string) ([]byte, bool, error)
	Close() error
}

// NewFromConfig builds a Source from cfg. The cfg.Type field selects the
// backend; required fields per type are documented on Config.
func NewFromConfig(ctx context.Context, cfg Config) (Source, error) {
	switch cfg.Type {
	case "local":
		return NewLocal(cfg.LocalDir)
	case "github":
		return NewGitHub(ctx, GitHubConfig{
			Repo:            cfg.GitHubRepo,
			Ref:             cfg.GitHubRef,
			ManifestPath:    cfg.GitHubManifestPath,
			BaseURL:         cfg.GitHubBaseURL,
			RefreshInterval: cfg.GitHubRefreshInterval,
		})
	default:
		return nil, fmt.Errorf("blobsource: unsupported type %q", cfg.Type)
	}
}
