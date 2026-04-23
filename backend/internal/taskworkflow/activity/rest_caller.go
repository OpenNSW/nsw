package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/OpenNSW/nsw/pkg/jsonutils"
	"github.com/OpenNSW/nsw/pkg/remote"
)

type RESTCaller struct {
	remoteManager *remote.Manager
}

func NewRESTCaller(remoteManager *remote.Manager) *RESTCaller {
	return &RESTCaller{
		remoteManager: remoteManager,
	}
}

func (a *RESTCaller) Name() ActivityType {
	return ActivityTypeRESTCaller
}

type RESTCallerConfig struct {
	ServiceID string         `json:"serviceId,omitempty"`
	Method    string         `json:"method,omitempty"`
	Path      string         `json:"path,omitempty"`
	Query     map[string]any `json:"query,omitempty"`
	Headers   map[string]any `json:"headers,omitempty"`
	Body      any            `json:"body,omitempty"`
	OutputKey string         `json:"outputKey,omitempty"`
}

func (a *RESTCaller) Execute(ctx context.Context, request Request) (*Result, error) {
	if a.remoteManager == nil {
		return nil, fmt.Errorf("rest caller: remote manager is required")
	}

	serviceID, req, outputKey, err := a.buildRequest(request)
	if err != nil {
		return nil, err
	}

	// 4. Perform the API call
	var responseData any
	if err := a.remoteManager.Call(ctx, serviceID, req, &responseData); err != nil {
		return nil, fmt.Errorf("rest caller: api call failed: %w", err)
	}

	// 5. Return the result to be written back to global context
	return &Result{
		Outputs: map[string]any{
			outputKey: responseData,
		},
		Status: StatusSucceeded,
	}, nil
}

func (a *RESTCaller) buildRequest(request Request) (string, remote.Request, string, error) {
	var cfg RESTCallerConfig

	configBytes, _ := json.Marshal(request.Config)
	if len(configBytes) == 0 || string(configBytes) == "null" {
		return "", remote.Request{}, "", fmt.Errorf("rest caller: config is required")
	}

	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return "", remote.Request{}, "", fmt.Errorf("rest caller: failed to parse config: %w", err)
	}

	if cfg.Path == "" {
		return "", remote.Request{}, "", fmt.Errorf("rest caller: path is required")
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.OutputKey == "" {
		cfg.OutputKey = "api_response"
	}

	lookup := func(key string) any {
		return request.Inputs[key]
	}

	resolvedPath := jsonutils.ResolveTemplateWithPlaceholders(cfg.Path, lookup)
	path, ok := resolvedPath.(string)
	if !ok {
		path = fmt.Sprint(resolvedPath)
	}

	resolvedQuery, err := resolveStringMap(jsonutils.ResolveTemplateWithPlaceholders(cfg.Query, lookup))
	if err != nil {
		return "", remote.Request{}, "", fmt.Errorf("rest caller: resolve query: %w", err)
	}

	resolvedHeaders, err := resolveStringMap(jsonutils.ResolveTemplateWithPlaceholders(cfg.Headers, lookup))
	if err != nil {
		return "", remote.Request{}, "", fmt.Errorf("rest caller: resolve headers: %w", err)
	}

	req := remote.Request{
		Method:  strings.ToUpper(cfg.Method),
		Path:    path,
		Query:   toURLValues(resolvedQuery),
		Headers: resolvedHeaders,
		Retry:   &remote.DefaultRetryConfig,
	}

	if cfg.Body != nil {
		req.Body = jsonutils.ResolveTemplateWithPlaceholders(cfg.Body, lookup)
	}

	return cfg.ServiceID, req, cfg.OutputKey, nil
}

func resolveStringMap(value any) (map[string]string, error) {
	if value == nil {
		return nil, nil
	}

	resolvedMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", value)
	}

	out := make(map[string]string, len(resolvedMap))
	for key, rawValue := range resolvedMap {
		out[key] = fmt.Sprint(rawValue)
	}

	return out, nil
}

func toURLValues(values map[string]string) url.Values {
	if len(values) == 0 {
		return nil
	}

	query := make(url.Values, len(values))
	for key, value := range values {
		query.Set(key, value)
	}

	return query
}
