package temporal

import (
	"log/slog"
	"strings"

	"github.com/OpenNSW/nsw/internal/config"
	"go.temporal.io/sdk/client"
	temporallog "go.temporal.io/sdk/log"
)

// NewClient creates a shared Temporal client for all workflow runtimes.
func NewClient(cfg config.TemporalConfig) (client.Client, error) {
	return client.Dial(optionsFromConfig(cfg))
}

func optionsFromConfig(cfg config.TemporalConfig) client.Options {
	hostPort := strings.TrimSpace(cfg.HostPort)
	if hostPort == "" {
		hostPort = "localhost:7233"
	}

	namespace := strings.TrimSpace(cfg.Namespace)
	if namespace == "" {
		namespace = "default"
	}

	return client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
		Logger:    temporallog.NewStructuredLogger(slog.Default()),
	}
}
