package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/pkg/remote"
)

// EventWaitPlugin replaces nsw-task-flow's register_task_and_wait plugin.
// Same shape — registers the task with an external queue and transitions to
// WAITING_FOR_EVENT — but uses the rich body envelope so the receiver can
// route by taskCode and call back via serviceUrl.
type EventWaitPlugin struct {
	client *dispatchHelper
}

func NewEventWaitPlugin(manager *remote.Manager, backendBaseURL string, devMode bool) *EventWaitPlugin {
	return &EventWaitPlugin{client: newDispatchHelper(manager, backendBaseURL, devMode)}
}

func (p *EventWaitPlugin) Name() string { return "register_task_and_wait" }

type eventWaitConfig struct {
	ServiceID string `json:"service_id"`
	Path      string `json:"path"`
	TaskCode  string `json:"task_code,omitempty"`
	TaskType  string `json:"task_type,omitempty"`
}

func (p *EventWaitPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg eventWaitConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("event_wait: invalid config: %w", err)
	}
	if cfg.ServiceID == "" {
		return fmt.Errorf("event_wait: service_id is required")
	}
	if cfg.Path == "" {
		return fmt.Errorf("event_wait: path is required")
	}

	ctx.Record.Status = "WAITING_FOR_EVENT"

	body := buildSubmissionBody(ctx.Record, &cfg.TaskCode, p.client.callbackTasksURL())
	if cfg.TaskType != "" {
		body["externalTaskType"] = cfg.TaskType
	}

	slog.Info("taskv2 event_wait: registering with external queue",
		"taskId", ctx.Record.TaskID, "serviceId", cfg.ServiceID, "path", cfg.Path, "taskCode", cfg.TaskCode, "taskType", cfg.TaskType)

	if err := p.client.post(ctx.Context, cfg.ServiceID, cfg.Path, body); err != nil {
		return err
	}
	return ErrSuspended
}
