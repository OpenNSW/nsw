package activity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRESTCallerBuildRequest(t *testing.T) {
	t.Run("resolves dynamic path query headers and body", func(t *testing.T) {
		caller := NewRESTCaller(nil)

		serviceID, req, outputKey, err := caller.buildRequest(Request{
			Config: map[string]any{
				"serviceId": "test-service",
				"method":    "POST",
				"path":      "/api/workflows/${workflow_id}/tasks/${task_id}",
				"query": map[string]any{
					"include": "$include",
					"page":    "$page",
				},
				"headers": map[string]any{
					"X-Workflow-Id": "$workflow_id",
				},
				"body": map[string]any{
					"request": map[string]any{
						"id":    "$task_id",
						"label": "task-${page}",
					},
				},
				"outputKey": "api_response",
			},
			Inputs: map[string]any{
				"workflow_id": "wf-123",
				"task_id":     "task-456",
				"include":     "details",
				"page":        2,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "test-service", serviceID)
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/api/workflows/wf-123/tasks/task-456", req.Path)
		assert.Equal(t, "details", req.Query.Get("include"))
		assert.Equal(t, "2", req.Query.Get("page"))
		assert.Equal(t, "wf-123", req.Headers["X-Workflow-Id"])
		assert.Equal(t, map[string]any{
			"request": map[string]any{
				"id":    "task-456",
				"label": "task-2",
			},
		}, req.Body)
		assert.Equal(t, "api_response", outputKey)
		assert.NotNil(t, req.Retry)
	})

	t.Run("applies defaults and allows omitted optional fields", func(t *testing.T) {
		caller := NewRESTCaller(nil)

		serviceID, req, outputKey, err := caller.buildRequest(Request{
			Config: map[string]any{
				"path": "/health",
			},
		})

		require.NoError(t, err)
		assert.Empty(t, serviceID)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "/health", req.Path)
		assert.Nil(t, req.Query)
		assert.Empty(t, req.Headers)
		assert.Nil(t, req.Body)
		assert.Equal(t, "api_response", outputKey)
		assert.NotNil(t, req.Retry)
	})

	t.Run("stringifies query and headers values", func(t *testing.T) {
		caller := NewRESTCaller(nil)

		_, req, _, err := caller.buildRequest(Request{
			Config: map[string]any{
				"path": "/search",
				"query": map[string]any{
					"page":   "$page",
					"active": "$active",
				},
				"headers": map[string]any{
					"X-Page":   "$page",
					"X-Active": "$active",
				},
			},
			Inputs: map[string]any{
				"page":   3,
				"active": true,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "3", req.Query.Get("page"))
		assert.Equal(t, "true", req.Query.Get("active"))
		assert.Equal(t, "3", req.Headers["X-Page"])
		assert.Equal(t, "true", req.Headers["X-Active"])
	})

	t.Run("requires path in config", func(t *testing.T) {
		caller := NewRESTCaller(nil)

		_, _, _, err := caller.buildRequest(Request{
			Config: map[string]any{
				"serviceId": "test-service",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "path is required")
	})

	t.Run("returns error when query config is not an object", func(t *testing.T) {
		caller := NewRESTCaller(nil)

		_, _, _, err := caller.buildRequest(Request{
			Config: map[string]any{
				"path":  "/search",
				"query": "invalid",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config")
		assert.Contains(t, err.Error(), "cannot unmarshal string")
	})

	t.Run("returns error when headers config is not an object", func(t *testing.T) {
		caller := NewRESTCaller(nil)

		_, _, _, err := caller.buildRequest(Request{
			Config: map[string]any{
				"path":    "/search",
				"headers": []any{"invalid"},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config")
		assert.Contains(t, err.Error(), "cannot unmarshal array")
	})
}

func TestRESTCallerExecuteRequiresConfig(t *testing.T) {
	_, err := NewRESTCaller(nil).Execute(context.Background(), Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote manager is required")
}

func TestRESTCallerExecuteRequiresRemoteManager(t *testing.T) {
	_, err := NewRESTCaller(nil).Execute(context.Background(), Request{
		Config: map[string]any{
			"path": "/health",
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote manager is required")
}

func TestResolveStringMap(t *testing.T) {
	t.Run("returns nil for nil value", func(t *testing.T) {
		result, err := resolveStringMap(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("stringifies object values", func(t *testing.T) {
		result, err := resolveStringMap(map[string]any{
			"page":   2,
			"active": true,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"page":   "2",
			"active": "true",
		}, result)
	})

	t.Run("returns error for non-object value", func(t *testing.T) {
		_, err := resolveStringMap("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected object")
	})
}

func TestToURLValues(t *testing.T) {
	t.Run("returns nil for empty map", func(t *testing.T) {
		assert.Nil(t, toURLValues(nil))
		assert.Nil(t, toURLValues(map[string]string{}))
	})

	t.Run("converts map to url values", func(t *testing.T) {
		values := toURLValues(map[string]string{
			"page":   "2",
			"active": "true",
		})

		require.NotNil(t, values)
		assert.Equal(t, "2", values.Get("page"))
		assert.Equal(t, "true", values.Get("active"))
	})
}
