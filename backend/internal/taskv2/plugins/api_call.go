package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/pkg/remote"
)

// APICallPlugin implements the "generic_api_call" FIRE_AND_FORGET task type.
// It fires the submission envelope at the configured service+path in a
// background goroutine (no wait for response) and immediately sets status to
// DISPATCHED so the workflow can continue.
type APICallPlugin struct {
	client *dispatchHelper
}

func NewAPICallPlugin(manager *remote.Manager, backendBaseURL string, devMode bool) *APICallPlugin {
	return &APICallPlugin{client: newDispatchHelper(manager, backendBaseURL, devMode)}
}

func (p *APICallPlugin) Name() string { return "generic_api_call" }

type apiCallConfig struct {
	ServiceID string `json:"service_id"`
	Path      string `json:"path"`
}

func (p *APICallPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg apiCallConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("api_call: invalid config: %w", err)
	}
	if cfg.ServiceID == "" {
		return fmt.Errorf("api_call: service_id is required")
	}
	if cfg.Path == "" {
		return fmt.Errorf("api_call: path is required")
	}

	ctx.Record.Status = "DISPATCHED"

	body := buildSubmissionBody(ctx.Record, nil, p.client.callbackTasksURL())

	slog.Info("taskv2 api_call: firing to external API (fire-and-forget)",
		"taskId", ctx.Record.TaskID, "serviceId", cfg.ServiceID, "path", cfg.Path)

	p.client.postAsync(cfg.ServiceID, cfg.Path, body)
	return nil
}
