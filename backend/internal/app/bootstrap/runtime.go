// Package runtime wires the nsw-task-flow TaskManager, plugin registry,
// template registry, and the two Temporal managers (parent + task) needed
// for hierarchical workflow execution.
//
// Architecture:
//
//	[Parent Workflow Temporal Manager]   queue: nsw-parent-workflow-queue
//	            │ activates TASK node
//	            ▼
//	      tm.StartTask  →  spins up a Task workflow on
//	[Task Workflow Temporal Manager]     queue: nsw-task-workflow-queue
//	            │ activates step
//	            ▼
//	      tm.StartSubTask  →  routes to a registered plugin
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	wf "github.com/OpenNSW/go-temporal-workflow"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw-task-flow/plugins"
	twfstore "github.com/OpenNSW/nsw-task-flow/store"

	customplugins "github.com/OpenNSW/nsw/internal/taskv2/plugins"
	"github.com/OpenNSW/nsw/pkg/remote"

	"go.temporal.io/sdk/client"
)

const (
	parentTaskQueue = "nsw-parent-workflow-queue"
	taskTaskQueue   = "nsw-task-workflow-queue"
)

// Runtime owns the lifecycle of the parent + task Temporal managers and the
// nsw-task-flow TaskManager built on top of them.
type Runtime struct {
	tm          *orchestrator.TaskManager
	parent      wf.TemporalManager
	task        wf.TemporalManager
	registry    *orchestrator.TaskTemplateRegistry
	pluginsRepo *plugins.Registry
}

// UpstreamService is notified when a parent workflow completes.
type UpstreamService interface {
	CompletionHandler(workflowID string, finalContext map[string]any) error
}

// Config bundles construction inputs for NewRuntime.
type Config struct {
	TemporalClient  client.Client
	Store           twfstore.TaskStore
	Registry        *orchestrator.TaskTemplateRegistry
	RemoteManager   *remote.Manager // resolves service base URLs from services.json
	BackendBaseURL  string          // public base URL of THIS backend, used in callbacks (e.g. http://localhost:8080)
	DevMode         bool            // if true, plugin dispatch swallows external HTTP errors
	UpstreamService UpstreamService // optional; called when a parent workflow completes
}

// NewRuntime constructs the parent + task Temporal managers, registers the
// six built-in nsw-task-flow plugins, and starts both Temporal workers.
func NewRuntime(cfg Config) (*Runtime, error) {
	if cfg.TemporalClient == nil {
		return nil, fmt.Errorf("taskv2 runtime: temporal client is required")
	}
	if cfg.Store == nil {
		return nil, fmt.Errorf("taskv2 runtime: task store is required")
	}
	if cfg.Registry == nil {
		return nil, fmt.Errorf("taskv2 runtime: template registry is required")
	}
	if cfg.RemoteManager == nil {
		return nil, fmt.Errorf("taskv2 runtime: remote manager is required")
	}

	// Custom dispatching plugins read from ctx.Record directly (so the body
	// they POST reflects the active sub-step's freshly-set ReviewerFormID,
	// SubTaskNodeID, etc.) and emit the SimpleForm-compatible envelope the
	// NPQS / FCAU OGA portals expect at /api/oga/inject.
	pluginsRepo := plugins.NewRegistry()
	registrations := []struct {
		taskType string
		plugin   plugins.TaskPlugin
	}{
		{"APPLICATION", plugins.NewUserInputPlugin()},
		{"APPLICATION", customplugins.NewExternalReviewPlugin(cfg.RemoteManager, cfg.BackendBaseURL, cfg.DevMode)},
		{"APPLICATION", customplugins.NewOfficerInputPlugin(cfg.RemoteManager, cfg.BackendBaseURL, cfg.DevMode)},
		{"WAIT_FOR_EVENT", customplugins.NewEventWaitPlugin(cfg.RemoteManager, cfg.BackendBaseURL, cfg.DevMode)},
		{"PAYMENT", customplugins.NewPaymentPlugin(cfg.RemoteManager, cfg.BackendBaseURL, cfg.DevMode)},
		{"FIRE_AND_FORGET", customplugins.NewAPICallPlugin(cfg.RemoteManager, cfg.BackendBaseURL, cfg.DevMode)},
	}
	for _, r := range registrations {
		if err := pluginsRepo.Register(r.taskType, r.plugin); err != nil {
			return nil, fmt.Errorf("taskv2 runtime: register plugin %q for %q: %w", r.plugin.Name(), r.taskType, err)
		}
	}

	// We need both Temporal managers + the TaskManager to refer to each other.
	// Resolve the cycle by capturing tm via closure and assigning it after construction.
	var tm *orchestrator.TaskManager

	// go-temporal-workflow v0.3.3 changed the handler signature to return
	// (map[string]any, error). The dispatch is asynchronous — TaskManager
	// drives the sub-workflow and wakes the parent via TaskDone — so we
	// return ErrResultPending after a successful kick-off so the engine
	// pauses the activity instead of completing it synchronously.
	parentTaskHandler := func(payload wf.TaskPayload) (map[string]any, error) {
		slog.Info("taskv2 parent: TASK node activated", "node", payload.NodeID, "template", payload.TaskTemplateID)
		if tm == nil {
			return nil, fmt.Errorf("taskv2 parent handler invoked before TaskManager was wired")
		}
		return tm.StartTask(payload)
	}
	parentCompletion := func(workflowID string, finalVariables map[string]any) error {
		slog.Info("taskv2 parent: workflow completed", "workflowId", workflowID, "finalVariables", finalVariables)
		if cfg.UpstreamService != nil {
			if err := cfg.UpstreamService.CompletionHandler(workflowID, finalVariables); err != nil {
				return fmt.Errorf("taskv2 parent: upstream completion handler: %w", err)
			}
		}
		return nil
	}
	parent := wf.NewTemporalManager(cfg.TemporalClient, parentTaskQueue, parentTaskHandler, parentCompletion)

	taskHandler := func(payload wf.TaskPayload) (map[string]any, error) {
		slog.Info("taskv2 task: step activated", "node", payload.NodeID, "template", payload.TaskTemplateID)
		if tm == nil {
			return nil, fmt.Errorf("taskv2 task handler invoked before TaskManager was wired")
		}
		data, err := tm.StartSubTask(payload)
		if err == nil {
			// Complete the acticity
			completionErr := tm.HandleTaskCompletion(payload.WorkflowID, data)
			if completionErr != nil {
				slog.Error("taskv2 task: failed to handle task completion", "error", completionErr)
			}
		}
		return data, err
	}
	taskCompletion := func(workflowID string, finalVariables map[string]any) error {
		slog.Info("taskv2 task: sub-workflow completed", "workflowId", workflowID)
		if tm == nil {
			return nil
		}
		// Merge the sub-workflow's final variables (userform, reviewerform, …) into
		// the task record BEFORE HandleTaskCompletion sets Status=COMPLETED. The library
		// only updates Status on completion and does not propagate finalVariables back
		// into record.Data, so without this step the trader portal would find empty
		// form data on any completed task.
		if rec, ok := cfg.Store.GetTaskByWorkflowID(workflowID); ok {
			if rec.Data == nil {
				rec.Data = make(map[string]any)
			}
			maps.Copy(rec.Data, finalVariables)
			cfg.Store.SaveTask(rec)
		}
		return tm.HandleTaskCompletion(workflowID, finalVariables)
	}
	taskMgr := wf.NewTemporalManager(cfg.TemporalClient, taskTaskQueue, taskHandler, taskCompletion)

	onTaskCompleted := func(parentWorkflowID, parentRunID, parentNodeID string, finalVariables map[string]any) error {
		slog.Info("taskv2: waking parent workflow", "parentWorkflowId", parentWorkflowID, "node", parentNodeID)
		return parent.TaskDone(context.Background(), parentWorkflowID, parentRunID, parentNodeID, finalVariables)
	}

	tm = orchestrator.NewTaskManager(cfg.Store, cfg.Registry, pluginsRepo, taskMgr, onTaskCompleted)

	if err := parent.StartWorker(); err != nil {
		return nil, fmt.Errorf("taskv2 runtime: start parent worker: %w", err)
	}
	if err := taskMgr.StartWorker(); err != nil {
		parent.StopWorker()
		return nil, fmt.Errorf("taskv2 runtime: start task worker: %w", err)
	}

	return &Runtime{
		tm:          tm,
		parent:      parent,
		task:        taskMgr,
		registry:    cfg.Registry,
		pluginsRepo: pluginsRepo,
	}, nil
}

// TaskManager returns the underlying TaskManager.
func (r *Runtime) TaskManager() *orchestrator.TaskManager { return r.tm }

// WorkflowManager returns the parent Temporal manager (used by the start-workflow handler).
func (r *Runtime) WorkflowManager() wf.TemporalManager { return r.parent }

// Registry returns the template registry (for the HTTP router).
func (r *Runtime) Registry() *orchestrator.TaskTemplateRegistry { return r.registry }

// Close stops both Temporal workers.
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	if r.task != nil {
		r.task.StopWorker()
	}
	if r.parent != nil {
		r.parent.StopWorker()
	}
	return nil
}
