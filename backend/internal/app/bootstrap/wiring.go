package bootstrap

import (
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	workflowmanager "github.com/OpenNSW/nsw/internal/workflow/manager"
)

// WireManagers wires Workflow Manager <-> Task Manager callbacks.
func WireManagers(wm workflowmanager.Manager, tm taskManager.TaskManager) error {
	// TODO: FIX ME
	// if wm == nil {
	// 	return fmt.Errorf("workflow manager cannot be nil")
	// }
	// if tm == nil {
	// 	return fmt.Errorf("task manager cannot be nil")
	// }

	// if err := wm.RegisterTaskHandler(tm.InitTask); err != nil {
	// 	return fmt.Errorf("failed to register task init callback: %w", err)
	// }

	// tm.RegisterUpstreamCallback(func(ctx context.Context, taskID string, state *plugin.State, extendedState *string, appendGlobalContext map[string]any, outcome *string) {
	// 	notification := taskManager.WorkflowManagerNotification{
	// 		TaskID:              taskID,
	// 		UpdatedState:        state,
	// 		ExtendedState:       extendedState,
	// 		AppendGlobalContext: appendGlobalContext,
	// 		Outcome:             outcome,
	// 	}

	// 	if err := wm.HandleTaskUpdate(ctx, notification); err != nil {
	// 		slog.ErrorContext(ctx, "workflow manager notification handling failed",
	// 			"taskID", taskID,
	// 			"error", err,
	// 		)
	// 	}
	// })

	return nil
}
