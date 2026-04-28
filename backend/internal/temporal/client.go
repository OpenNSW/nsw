package temporal

import (
	"log/slog"

	"github.com/OpenNSW/nsw/internal/config"
	"go.temporal.io/sdk/client"
	temporallog "go.temporal.io/sdk/log"
)

// NewClient creates a shared Temporal client for all workflow runtimes.
func NewClient(cfg config.TemporalConfig) (client.Client, error) {
	return client.Dial(optionsFromConfig(cfg))
}

func optionsFromConfig(cfg config.TemporalConfig) client.Options {
	return client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
		Logger:    temporallog.NewStructuredLogger(slog.Default()),
	}
}
