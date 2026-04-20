package temporal

import (
	"log/slog"

	"go.temporal.io/sdk/client"
	temporallog "go.temporal.io/sdk/log"
)

// NewClient creates a shared Temporal client for all workflow runtimes.
func NewClient() (client.Client, error) {
	// TODO: Make Temporal connection settings configurable via config.Config
	// (e.g., host:port and namespace) instead of relying on SDK defaults.
	return client.Dial(client.Options{
		Logger: temporallog.NewStructuredLogger(slog.Default()),
	})
}
