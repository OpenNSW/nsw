package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/OpenNSW/nsw/internal/taskworkflow/activity"
	"github.com/OpenNSW/nsw/pkg/remote"
)

func TestRESTCallerExecuteIntegration(t *testing.T) {
	if os.Getenv("ENABLE_ACTIVITY_INTEGRATION_TESTS") != "1" {
		t.Skip("set ENABLE_ACTIVITY_INTEGRATION_TESTS=1 to run activity integration tests")
	}

	t.Run("dispatches resolved request through remote manager", func(t *testing.T) {
		var capturedPath string
		var capturedQuery string
		var capturedHeader string
		var capturedBody map[string]any

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			capturedQuery = r.URL.RawQuery
			capturedHeader = r.Header.Get("X-Workflow-Id")

			if r.Body != nil {
				defer r.Body.Close()
				require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
			}

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"ok":         true,
				"request_id": "req-123",
			}))
		}))
		defer server.Close()

		registryPath := filepath.Join(t.TempDir(), "services.json")
		require.NoError(t, os.WriteFile(registryPath, []byte(`{
  "version": "1.0",
  "services": [
    {
      "id": "test-service",
      "url": "`+server.URL+`",
      "timeout": "5s"
    }
  ]
}`), 0o600))

		rm := remote.NewManager()
		require.NoError(t, rm.LoadServices(registryPath))

		caller := activity.NewRESTCaller(rm)
		result, err := caller.Execute(context.Background(), activity.Request{
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
		require.NotNil(t, result)
		assert.Equal(t, activity.StatusSucceeded, result.Status)
		assert.Equal(t, "/api/workflows/wf-123/tasks/task-456", capturedPath)
		assert.Equal(t, "include=details&page=2", capturedQuery)
		assert.Equal(t, "wf-123", capturedHeader)
		assert.Equal(t, map[string]any{
			"request": map[string]any{
				"id":    "task-456",
				"label": "task-2",
			},
		}, capturedBody)
		assert.Equal(t, map[string]any{
			"ok":         true,
			"request_id": "req-123",
		}, result.Outputs["api_response"])
	})
}
