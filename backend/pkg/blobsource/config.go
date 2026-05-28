package blobsource

import (
	"fmt"
	"time"

	"github.com/OpenNSW/nsw/internal/validation"
)

// Config selects and configures a blob source backend.
//
// Type must be either "local" or "github". The fields required for each
// backend are documented on the fields themselves and enforced by Validate.
type Config struct {
	Type string // "local" or "github"

	// local backend
	LocalDir string

	// github backend
	GitHubRepo            string
	GitHubRef             string
	GitHubBaseURL         string        // optional; defaults to DefaultGitHubBaseURL
	GitHubRefreshInterval time.Duration // optional; 0 disables background refresh
}

func (c Config) Validate() error {
	switch c.Type {
	case "local":
		if c.LocalDir == "" {
			return fmt.Errorf("BLOBSOURCE_LOCAL_DIR is required when BLOBSOURCE_TYPE=local")
		}
	case "github":
		if c.GitHubRepo == "" {
			return fmt.Errorf("BLOBSOURCE_GITHUB_REPO is required when BLOBSOURCE_TYPE=github")
		}
		if c.GitHubRef == "" {
			return fmt.Errorf("BLOBSOURCE_GITHUB_REF is required when BLOBSOURCE_TYPE=github")
		}
		if c.GitHubBaseURL != "" {
			if err := validation.HTTPURL("BLOBSOURCE_GITHUB_BASE_URL", c.GitHubBaseURL); err != nil {
				return err
			}
		}
		if c.GitHubRefreshInterval < 0 {
			return fmt.Errorf("BLOBSOURCE_GITHUB_REFRESH_INTERVAL cannot be negative")
		}
	default:
		return fmt.Errorf("unsupported BLOBSOURCE_TYPE: %q", c.Type)
	}
	return nil
}
