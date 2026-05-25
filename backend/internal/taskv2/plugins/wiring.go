// Package plugins wires the native nsw-task-flow plugins into a plugin
// registry. The taskType keys must match the Type field on SubTaskTemplate
// configs loaded by internal/taskv2/registry.
package plugins

import (
	"fmt"

	flowplugins "github.com/OpenNSW/nsw-task-flow/plugins"
	"github.com/OpenNSW/nsw/pkg/remote"
)

// Task type keys. These must match the SubTaskTemplate.Type values declared
// in the JSON configs that internal/taskv2/registry.LoadConfigsInto reads.
const (
	TaskTypeUserInput      = "USER_INPUT"
	TaskTypeExternalReview = "EXTERNAL_REVIEW"
	TaskTypePayment        = "PAYMENT"
	TaskTypeAPICall        = "API_CALL"
)

// Register installs the taskv2 plugins on reg.
//
// EXTERNAL_REVIEW uses our local plugin (ExternalReviewPlugin) that resolves
// targets via remote.Manager and posts the OGA submission envelope. Payment
// and API_CALL still use the library's stock plugins with the default HTTP
// dispatcher — swap them out as local replacements are written.
func Register(reg *flowplugins.Registry, mgr *remote.Manager, backendBaseURL string, devMode bool) error {
	if reg == nil {
		return fmt.Errorf("plugins: registry is nil")
	}
	if mgr == nil {
		return fmt.Errorf("plugins: remote manager is nil")
	}

	entries := []struct {
		taskType string
		plugin   flowplugins.TaskPlugin
	}{
		{TaskTypeUserInput, flowplugins.NewUserInputPlugin()},
		{TaskTypeExternalReview, NewExternalReviewPlugin(mgr, backendBaseURL, devMode)},
		{TaskTypePayment, flowplugins.NewPaymentPlugin(flowplugins.DefaultHTTPDispatcher)},
		{TaskTypeAPICall, flowplugins.NewAPICallPlugin(flowplugins.DefaultHTTPDispatcher)},
	}

	for _, e := range entries {
		if err := reg.Register(e.taskType, e.plugin); err != nil {
			return fmt.Errorf("plugins: register %s: %w", e.taskType, err)
		}
	}
	return nil
}
