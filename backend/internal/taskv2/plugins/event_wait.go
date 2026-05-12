package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// EventWaitPlugin replaces nsw-task-flow's register_task_and_wait plugin.
// Same shape — registers the task with an external queue and transitions to
// WAITING_FOR_EVENT — but uses the rich body envelope so the receiver can
// route by taskCode and call back via serviceUrl.
type EventWaitPlugin struct {
	client *dispatchClient
}

func NewEventWaitPlugin(backendBaseURL string, devMode bool) *EventWaitPlugin {
	return &EventWaitPlugin{client: newDispatchClient(backendBaseURL, devMode)}
}

func (p *EventWaitPlugin) Name() string { return "register_task_and_wait" }

type eventWaitConfig struct {
	ExternalURL string `json:"external_url"`
	TaskCode    string `json:"task_code,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
}

func (p *EventWaitPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg eventWaitConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("event_wait: invalid config: %w", err)
	}
	if cfg.ExternalURL == "" {
		return fmt.Errorf("event_wait: external_url is required")
	}

	ctx.Record.Status = "WAITING_FOR_EVENT"

	body := buildSubmissionBody(ctx.Record, &cfg.TaskCode, p.client.callbackTasksURL())
	if cfg.TaskType != "" {
		body["externalTaskType"] = cfg.TaskType
	}

	slog.Info("taskv2 event_wait: registering with external queue",
		"taskId", ctx.Record.TaskID, "url", cfg.ExternalURL, "taskCode", &cfg.TaskCode, "taskType", cfg.TaskType)

	if err := p.client.post(ctx.Context, cfg.ExternalURL, body); err != nil {
		return err
	}
	return ErrSuspended
}
