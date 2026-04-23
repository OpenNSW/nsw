package activity

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONBuilderBuild(t *testing.T) {
	t.Run("builds json from input data", func(t *testing.T) {
		builder := NewJSONBuilder()

		result, err := builder.Build(context.Background(), JSONBuilderInput{
			Data: map[string]any{
				"name":  "task-workflow",
				"count": 2,
				"meta": map[string]any{
					"enabled": true,
				},
			},
		})

		require.NoError(t, err)
		assert.JSONEq(t, `{"count":2,"meta":{"enabled":true},"name":"task-workflow"}`, string(result.JSON))
	})

	t.Run("returns error when input cannot be marshalled", func(t *testing.T) {
		builder := NewJSONBuilder()

		result, err := builder.Build(context.Background(), JSONBuilderInput{
			Data: map[string]any{
				"invalid": func() {},
			},
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "marshal json builder input")
	})
}

func TestJSONBuilderExecute(t *testing.T) {
	t.Run("uses resolved template from config", func(t *testing.T) {
		builder := NewJSONBuilder()

		result, err := builder.Execute(context.Background(), Request{
			Config: map[string]any{
				"template": map[string]any{
					"request": map[string]any{
						"id":      "$workflow_id",
						"name":    "$name",
						"version": "$version",
						"label":   "workflow-${version}",
						"meta": map[string]any{
							"enabled": "$enabled",
						},
					},
				},
			},
			Inputs: map[string]any{
				"workflow_id": "wf-1",
				"name":        "task-workflow",
				"version":     2,
				"enabled":     true,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		raw, ok := result.Outputs["json"].(json.RawMessage)
		require.True(t, ok)
		assert.JSONEq(t, `{"request":{"id":"wf-1","name":"task-workflow","version":2,"label":"workflow-2","meta":{"enabled":true}}}`, string(raw))
	})

	t.Run("supports custom output key", func(t *testing.T) {
		builder := NewJSONBuilder()

		result, err := builder.Execute(context.Background(), Request{
			Config: map[string]any{
				"outputKey": "request_body",
				"template": map[string]any{
					"id": "$task_id",
				},
			},
			Inputs: map[string]any{
				"task_id": "task-1",
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		raw, ok := result.Outputs["request_body"].(json.RawMessage)
		require.True(t, ok)
		assert.JSONEq(t, `{"id":"task-1"}`, string(raw))
	})

	t.Run("returns error when template is not configured", func(t *testing.T) {
		builder := NewJSONBuilder()

		result, err := builder.Execute(context.Background(), Request{
			Inputs: map[string]any{
				"name":    "task-workflow",
				"version": 2,
			},
		})

		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "template is required")
	})
}
