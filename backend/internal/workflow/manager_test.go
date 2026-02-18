package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func TestPluginStateToWorkflowNodeState(t *testing.T) {
	tests := []struct {
		name          string
		input         plugin.State
		expectedState model.WorkflowNodeState
		expectError   bool
	}{
		{
			name:          "InProgress",
			input:         plugin.InProgress,
			expectedState: model.WorkflowNodeStateInProgress,
			expectError:   false,
		},
		{
			name:          "Completed",
			input:         plugin.Completed,
			expectedState: model.WorkflowNodeStateCompleted,
			expectError:   false,
		},
		{
			name:          "Failed",
			input:         plugin.Failed,
			expectedState: model.WorkflowNodeStateFailed,
			expectError:   false,
		},
		{
			name:          "Unknown",
			input:         plugin.State("unknown"),
			expectedState: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pluginStateToWorkflowNodeState(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedState, result)
			}
		})
	}
}
