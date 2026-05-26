package taskv2

import (
	"context"
	"fmt"
	"log"

	engine "github.com/OpenNSW/go-temporal-workflow"
	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	"github.com/OpenNSW/nsw-task-flow/plugins"
	"github.com/OpenNSW/nsw/internal/payments"
	"github.com/OpenNSW/nsw/internal/taskv2/registry"
	taskrenderer "github.com/OpenNSW/nsw/internal/taskv2/renderer"
	"github.com/OpenNSW/nsw/internal/taskv2/store"
	"github.com/OpenNSW/nsw/pkg/uiprojector"
	"go.temporal.io/sdk/client"
	"gorm.io/gorm"
)

// WireResult bundles the taskv2 objects bootstrap needs to expose handlers
// and to wire the parent workflow runner.
type WireResult struct {
	Manager   *orchestrator.TaskManager
	Runner    engine.TemporalManager
	Store     *store.GormTaskStore
	Assembler *taskrenderer.ZoneViewAssembler
}

// WireTaskV2 builds and starts the taskv2 stack on MICRO_WORKFLOW_QUEUE.
// The returned TemporalManager runs the per-task (micro) sub-workflows; the
// parent/macro workflow runner is owned by the workflow package and must be
// wired separately. The onTaskCompleted callback is invoked when a task
// workflow finishes — typically to call TaskDone on the parent runner so the
// macro workflow can advance past its Task node. The plugin registry must be
// pre-populated by the caller; an empty registry means every sub-task
// activation will fail to find a handler.
func WireTaskV2(db *gorm.DB, c *client.Client, pluginsRegistry *plugins.Registry, paymentService payments.PaymentService, onTaskCompleted orchestrator.TaskCompletedCallback) (*WireResult, func() error, error) {
	if pluginsRegistry == nil {
		return nil, nil, fmt.Errorf("taskv2: plugins registry is nil")
	}

	taskStore := store.NewGormTaskStore(db)

	templateRegistry := registry.NewInMemRegistry()
	if err := registry.LoadConfigsInto(templateRegistry, "configs/fcau"); err != nil {
		return nil, nil, fmt.Errorf("taskv2: load configs: %w", err)
	}

	projectors := append(uiprojector.DefaultProjectors(), taskrenderer.NewPaymentProjector(paymentService))
	uiAssembler, err := uiprojector.NewAssembler(
		registryTemplateProvider{reg: templateRegistry},
		projectors,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("taskv2: build assembler: %w", err)
	}
	taskRenderer := taskrenderer.NewTaskRenderer(uiAssembler)
	zoneAssembler := taskrenderer.NewZoneViewAssembler(taskRenderer)

	var tm *orchestrator.TaskManager

	// Handlers for events on the per-task (micro) sub-workflows running on
	// MICRO_WORKFLOW_QUEUE. Nodes inside a task workflow activate subtasks
	// via tm.StartSubTask, which dispatches to the matching plugin.
	microActivationHandler := func(payload engine.TaskPayload) (map[string]any, error) {
		log.Printf("\n[Micro Workflow] SubTask activated: node=%s template=%s\n", payload.NodeID, payload.TaskTemplateID)
		if tm == nil {
			return nil, fmt.Errorf("task manager is not initialized (misconfiguration)")
		}
		return tm.StartSubTask(payload)
	}

	microCompletionHandler := func(workflowID string, finalVariables map[string]any) error {
		log.Printf("\n[Micro Workflow] Completed. Final state: %v\n", finalVariables)
		if tm == nil {
			return fmt.Errorf("task manager is not initialized (misconfiguration)")
		}
		return tm.HandleTaskCompletion(context.Background(), workflowID, finalVariables)
	}

	workflowRunner := engine.NewTemporalManager(*c, "MICRO_WORKFLOW_QUEUE", microActivationHandler, microCompletionHandler)

	tm = orchestrator.NewTaskManager(taskStore, templateRegistry, pluginsRegistry, workflowRunner, onTaskCompleted, taskRenderer)

	if err := workflowRunner.StartWorker(); err != nil {
		return nil, nil, fmt.Errorf("taskv2: start worker: %w", err)
	}

	stop := func() error {
		workflowRunner.StopWorker()
		return nil
	}

	return &WireResult{
		Manager:   tm,
		Runner:    workflowRunner,
		Store:     taskStore,
		Assembler: zoneAssembler,
	}, stop, nil
}

// registryTemplateProvider adapts the orchestrator's TaskTemplateRegistry to
// uiprojector's TemplateProvider contract. Generic templates (JSONForms
// schemas, markdown bodies, etc.) are resolved through GetGenericTemplate.
type registryTemplateProvider struct {
	reg orchestrator.TaskTemplateRegistry
}

func (p registryTemplateProvider) GetTemplate(_ context.Context, id string) ([]byte, error) {
	raw, ok := p.reg.GetGenericTemplate(id)
	if !ok {
		return nil, fmt.Errorf("template %q not found", id)
	}
	return []byte(raw), nil
}
