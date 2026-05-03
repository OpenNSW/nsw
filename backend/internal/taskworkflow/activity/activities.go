package activity

import (
	"context"

	taskPlugin "github.com/OpenNSW/nsw/internal/task/plugin"
)

type ActivityType = taskPlugin.Type

const (
	ActivityTypeJSONBuilder ActivityType = "JSON_BUILDER"
	ActivityTypeRESTCaller  ActivityType = "REST_CALLER"
)

type Status string

const (
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
	StatusRetryable Status = "RETRYABLE"
)

type Result struct {
	Outputs       map[string]any `json:"outputs,omitempty"`
	RenderPayload map[string]any `json:"renderPayload,omitempty"`
	Status        Status         `json:"status,omitempty"`
	PersistRender bool           `json:"persistRender,omitempty"`
	Message       string         `json:"message,omitempty"`
}

type Executor interface {
	Name() ActivityType
	Execute(ctx context.Context, request Request) (*Result, error)
}
