package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	workflowmanager "github.com/OpenNSW/go-temporal-workflow"

	taskmanager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/workflow/service"

	"go.temporal.io/sdk/client"
)

const (
	interpreterTaskQueue = "INTERPRETER_TASK_QUEUE"
	activationTimeout    = 30 * time.Second
)

type temporalManagerFactory func(
	activationHandler workflowmanager.TaskActivationHandler,
	completionHandler workflowmanager.WorkflowCompletionHandler,
) workflowmanager.TemporalManager

// Runtime owns Temporal workflow manager lifecycle for the application runtime.
type Runtime struct {
	manager       workflowmanager.TemporalManager
	runtimeCancel context.CancelFunc
}

// NewRuntime creates, wires, and starts the workflow runtime.
func NewRuntime(temporalClient client.Client, tm taskmanager.TaskManager, templateProvider service.TemplateProvider, upstreamService UpstreamService) (*Runtime, error) {
	if temporalClient == nil {
		return nil, fmt.Errorf("temporal client is required")
	}

	createManager := func(
		activationHandler workflowmanager.TaskActivationHandler,
		completionHandler workflowmanager.WorkflowCompletionHandler,
	) workflowmanager.TemporalManager {
		return workflowmanager.NewTemporalManager(
			temporalClient,
			interpreterTaskQueue,
			activationHandler,
			completionHandler,
		)
	}

	return newRuntimeWithFactory(tm, templateProvider, createManager, upstreamService)
}

func newRuntimeWithFactory(tm taskmanager.TaskManager, templateProvider service.TemplateProvider, createManager temporalManagerFactory, upstreamService UpstreamService) (*Runtime, error) {
	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())

	activationHandler := func(payload workflowmanager.TaskPayload) error {
		activationCtx, cancel := context.WithTimeout(runtimeCtx, activationTimeout)
		defer cancel()

		template, err := templateProvider.GetWorkflowNodeTemplateByID(activationCtx, payload.TaskTemplateID)
		if err != nil {
			return fmt.Errorf("error getting workflow node template: %w", err)
		}

		// TODO: We need to pass the TaskPayload.RunID in the future to avoid issues with
		// task retries. For example, when retrying a task instance, a stale version might
		// send a completion that will trigger the new version.
		tmRequest := taskmanager.InitTaskRequest{
			TaskID:                 payload.NodeID,
			WorkflowID:             payload.WorkflowID,
			WorkflowNodeTemplateID: template.ID,
			GlobalState:            payload.Inputs,
			Type:                   template.Type,
			Config:                 template.Config,
		}

		if _, err := tm.InitTask(activationCtx, tmRequest); err != nil {
			return fmt.Errorf("error initializing task manager: %w", err)
		}

		return nil
	}

	completionHandler := func(workflowID string, finalContext map[string]any) error {
		slog.Info("Workflow logically completed", "workflowID", workflowID, "finalContext", finalContext)

		if upstreamService != nil {
			if err := upstreamService.CompletionHandler(workflowID, finalContext); err != nil {
				return fmt.Errorf("error calling upstream completion handler: %w", err)
			}
		}

		return nil
	}

	workflowManager := createManager(activationHandler, completionHandler)

	if err := workflowManager.StartWorker(); err != nil {
		runtimeCancel()
		return nil, fmt.Errorf("failed to start workflow manager worker: %w", err)
	}

	taskDoneWrapper := func(ctx context.Context, workflowID string, taskID string, outputs map[string]any) {
		if err := workflowManager.TaskDone(ctx, workflowID, "", taskID, outputs); err != nil {
			slog.ErrorContext(ctx, "error completing task", "error", err)
		}
	}
	tm.RegisterUpstreamDoneCallback(taskDoneWrapper)

	return &Runtime{
		manager:       workflowManager,
		runtimeCancel: runtimeCancel,
	}, nil
}

// Manager returns the started workflow manager.
func (r *Runtime) Manager() workflowmanager.TemporalManager {
	if r == nil {
		return nil
	}
	return r.manager
}

// Close stops worker polling and cancels runtime-scoped contexts.
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}

	if r.manager != nil {
		r.manager.StopWorker()
	}
	if r.runtimeCancel != nil {
		r.runtimeCancel()
	}

	return nil
}
