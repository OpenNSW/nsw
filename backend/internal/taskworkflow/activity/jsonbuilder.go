package activity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/pkg/jsonutils"
)

type Request struct {
	WorkflowID string
	RunID      string
	NodeID     string
	Config     map[string]any
	Inputs     map[string]any
}

type JSONBuilderInput struct {
	Data any `json:"data"`
}

type JSONBuilderOutput struct {
	JSON json.RawMessage `json:"json"`
}

type JSONBuilderConfig struct {
	Template  any    `json:"template"`
	OutputKey string `json:"outputKey,omitempty"`
}

type JSONBuilder struct{}

func NewJSONBuilder() *JSONBuilder {
	return &JSONBuilder{}
}

func (a *JSONBuilder) Name() ActivityType {
	return ActivityTypeJSONBuilder
}

func (a *JSONBuilder) Build(_ context.Context, input JSONBuilderInput) (*JSONBuilderOutput, error) {
	data, err := json.Marshal(input.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal json builder input: %w", err)
	}

	return &JSONBuilderOutput{
		JSON: data,
	}, nil
}

func (a *JSONBuilder) Execute(ctx context.Context, request Request) (*Result, error) {
	cfg, err := parseJSONBuilderConfig(request.Config)
	if err != nil {
		return nil, err
	}
	if cfg.Template == nil {
		return nil, fmt.Errorf("json builder: template is required")
	}

	data := jsonutils.ResolveTemplateWithPlaceholders(cfg.Template, func(key string) any {
		return request.Inputs[key]
	})

	result, err := a.Build(ctx, JSONBuilderInput{Data: data})
	if err != nil {
		return nil, err
	}

	outputKey := cfg.OutputKey
	if outputKey == "" {
		outputKey = "json"
	}

	return &Result{
		Outputs: map[string]any{
			outputKey: result.JSON,
		},
		Status: StatusSucceeded,
	}, nil
}

func parseJSONBuilderConfig(raw map[string]any) (*JSONBuilderConfig, error) {
	if len(raw) == 0 {
		return &JSONBuilderConfig{}, nil
	}

	configBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal json builder config: %w", err)
	}

	var cfg JSONBuilderConfig
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return nil, fmt.Errorf("parse json builder config: %w", err)
	}

	return &cfg, nil
}
