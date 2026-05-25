package workflow

import (
	"fmt"
	"log/slog"

	engine "github.com/OpenNSW/go-temporal-workflow"
	"go.temporal.io/sdk/client"
)

const ParentWorkflowQueue = "INTERPRETER_TASK_QUEUE"

// TaskActivator is the narrow surface needed from a task manager when the
// parent workflow reaches a Task node. taskv2's orchestrator.TaskManager
// satisfies this via its StartTask method.
type TaskActivator interface {
	StartTask(payload engine.TaskPayload) (map[string]any, error)
}

// UpstreamService is the narrow surface needed to notify a downstream domain
// (e.g. consignment) when a parent workflow finishes. Implementors decide how
// to dispatch on workflowID and finalContext.
type UpstreamService interface {
	CompletionHandler(workflowID string, finalContext map[string]any) error
}

// WireParentRunner starts the Temporal worker that runs macro/parent workflows
// on ParentWorkflowQueue. When a parent workflow reaches a Task node, the
// activator's StartTask is invoked to spawn the corresponding task workflow.
// On parent-workflow completion, upstream.CompletionHandler is invoked so the
// owning domain (e.g. consignment) can advance its own state. Pass a nil
// upstream to opt out of the notification.
//
// The returned stop closure halts the worker and should be invoked during
// shutdown.
func WireParentRunner(c client.Client, activator TaskActivator, upstream UpstreamService) (engine.TemporalManager, func() error, error) {
	onActivation := func(payload engine.TaskPayload) (map[string]any, error) {
		return activator.StartTask(payload)
	}

	onCompletion := func(workflowID string, finalVariables map[string]any) error {
		slog.Info("parent workflow completed", "workflowID", workflowID, "finalVariables", finalVariables)
		if upstream != nil {
			if err := upstream.CompletionHandler(workflowID, finalVariables); err != nil {
				return fmt.Errorf("upstream completion handler: %w", err)
			}
		}
		return nil
	}

	runner := engine.NewTemporalManager(c, ParentWorkflowQueue, onActivation, onCompletion)
	if err := runner.StartWorker(); err != nil {
		return nil, nil, fmt.Errorf("workflow: start parent worker: %w", err)
	}

	stop := func() error {
		runner.StopWorker()
		return nil
	}

	return runner, stop, nil
}
