package internal

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/OpenNSW/nsw/oga/pkg/templatesource"
)

// NewFormSource builds a templatesource.Source for OGA's form loader. It maps
// OGA's env-driven Config onto templatesource's typed configs; the actual
// source implementations live in github.com/OpenNSW/nsw/oga/pkg/templatesource
// so other services / loaders (e.g. a future workflow loader) can reuse them.
//
// The package is consumer-agnostic — for forms, OGA points the local backend
// at <OGA_CONFIG_DIR>/forms; a different consumer would compose a different
// path.
func NewFormSource(ctx context.Context, cfg Config) (templatesource.Source, error) {
	switch cfg.FormSource {
	case "local":
		return templatesource.NewLocal(filepath.Join(cfg.ConfigDir, "forms"))
	case "github":
		return templatesource.NewGitHub(ctx, templatesource.GitHubConfig{
			Repo:            cfg.FormGitHubRepo,
			Ref:             cfg.FormGitHubRef,
			RefreshInterval: cfg.FormManifestRefreshInterval,
		})
	default:
		return nil, fmt.Errorf("unsupported OGA_FORM_SOURCE %q (expected \"local\" or \"github\")", cfg.FormSource)
	}
}
