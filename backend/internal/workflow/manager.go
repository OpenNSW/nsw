package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"gorm.io/gorm"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

// Manager is the refactored workflow manager that coordinates between services, routers, and task manager
type Manager struct {
	tm                     taskManager.TaskManager
	hsCodeService          *service.HSCodeService
	consignmentService     *service.ConsignmentService
	workflowNodeService    *service.WorkflowNodeService
	templateService        *service.TemplateService
	hsCodeRouter           *router.HSCodeRouter
	consignmentRouter      *router.ConsignmentRouter
	workflowNodeUpdateChan chan taskManager.WorkflowManagerNotification
	ctx                    context.Context
	cancel                 context.CancelFunc
}

// NewManager creates a new refactored workflow manager
func NewManager(tm taskManager.TaskManager, ch chan taskManager.WorkflowManagerNotification, db *gorm.DB) *Manager {
	// Initialize services
	hsCodeService := service.NewHSCodeService(db)
	workflowNodeService := service.NewWorkflowNodeService(db)
	templateService := service.NewTemplateService(db)
	consignmentService := service.NewConsignmentService(db, templateService, workflowNodeService)

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		tm:                     tm,
		hsCodeService:          hsCodeService,
		consignmentService:     consignmentService,
		workflowNodeService:    workflowNodeService,
		templateService:        templateService,
		workflowNodeUpdateChan: ch,
		ctx:                    ctx,
		cancel:                 cancel,
	}

	// Set pre-commit validation callback to ensure task registration happens within transaction
	consignmentService.SetPreCommitValidationCallback(m.registerWorkflowNodesWithTaskManager)

	// Initialize routers
	m.hsCodeRouter = router.NewHSCodeRouter(hsCodeService)
	m.consignmentRouter = router.NewConsignmentRouter(consignmentService, nil) // No longer need callback in router

	// Start listening for workflow node updates
	m.StartWorkflowNodeUpdateListener()

	return m
}

// StartWorkflowNodeUpdateListener starts a goroutine that listens for workflow node updates
func (m *Manager) StartWorkflowNodeUpdateListener() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				slog.Info("workflow node update listener stopped")
				return
			case update := <-m.workflowNodeUpdateChan:
				// Validate and convert plugin state to workflow node state
				if update.UpdatedState == nil {
					slog.Error("received nil state in workflow node update",
						"taskID", update.TaskID)
					continue
				}

				workflowState, err := pluginStateToWorkflowNodeState(*update.UpdatedState)
				if err != nil {
					slog.Error("invalid state in workflow node update",
						"taskID", update.TaskID,
						"pluginState", *update.UpdatedState,
						"error", err)
					continue
				}

				updateReq := model.UpdateWorkflowNodeDTO{
					WorkflowNodeID:      update.TaskID,
					State:               workflowState,
					AppendGlobalContext: update.AppendGlobalContext,
					ExtendedState:       update.ExtendedState,
				}

				newReadyNodes, newGlobalContext, err := m.consignmentService.UpdateWorkflowNodeStateAndPropagateChanges(m.ctx, &updateReq)
				if err != nil {
					slog.Error("failed to handle workflow node update",
						"taskID", update.TaskID,
						"state", workflowState,
						"extendedState", update.ExtendedState,
						"globalContext", newGlobalContext,
						"error", err)
					// TODO: Implement retry mechanism with exponential backoff
					// - Store failed update in persistent queue (failed_workflow_updates table)
					// - Add background worker to retry failed updates periodically
					// - Implement max retry limits and dead-letter queue for permanent failures
					continue
				}

				if len(newReadyNodes) > 0 {
					err := m.registerWorkflowNodesWithTaskManager(newReadyNodes, newGlobalContext)
					if err != nil {
						slog.Error("failed to register new ready nodes with task manager",
							"taskID", update.TaskID,
							"newReadyNodeCount", len(newReadyNodes),
							"error", err)
						// Continue processing even if registration fails
						// The nodes are already in READY state in DB
					}
				}
			}
		}
	}()
}

// StopWorkflowNodeUpdateListener stops the workflow node update listener
func (m *Manager) StopWorkflowNodeUpdateListener() {
	if m.cancel != nil {
		m.cancel()
	}
}

// registerWorkflowNodesWithTaskManager registers workflow nodes with the Task Manager
// This is called when new READY workflow nodes are created
// Returns an error if any task registration fails
func (m *Manager) registerWorkflowNodesWithTaskManager(workflowNodes []model.WorkflowNode, consignmentGlobalContext map[string]any) error {
	for _, node := range workflowNodes {
		nodeTemplate, err := m.templateService.GetWorkflowNodeTemplateByID(m.ctx, node.WorkflowNodeTemplateID)
		if err != nil {
			return fmt.Errorf("failed to get workflow node template %s: %w", node.WorkflowNodeTemplateID, err)
		}
		initTaskRequest := taskManager.InitTaskRequest{
			ConsignmentID: node.ConsignmentID,
			TaskID:        node.ID,
			StepID:        node.WorkflowNodeTemplateID.String(),
			Type:          nodeTemplate.Type,
			GlobalState:   consignmentGlobalContext,
			Config:        nodeTemplate.Config,
		}
		response, err := m.tm.InitTask(m.ctx, initTaskRequest)
		if err != nil {
			return fmt.Errorf("failed to initialize task in task manager for node %s: %w", node.ID, err)
		}
		slog.Info("successfully registered workflow node with task manager", "Response", response.Result)
	}
	return nil
}

// HTTP Handler delegation methods

// HandleGetAllHSCodes handles GET /api/v1/hscodes
func (m *Manager) HandleGetAllHSCodes(w http.ResponseWriter, r *http.Request) {
	m.hsCodeRouter.HandleGetAllHSCodes(w, r)
}

// HandleCreateConsignment handles POST /api/v1/consignments
func (m *Manager) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	m.consignmentRouter.HandleCreateConsignment(w, r)
}

// HandleGetConsignmentsByTraderID handles GET /api/v1/consignments?traderId={traderId}
func (m *Manager) HandleGetConsignmentsByTraderID(w http.ResponseWriter, r *http.Request) {
	m.consignmentRouter.HandleGetConsignmentsByTraderID(w, r)
}

// HandleGetConsignmentByID handles GET /api/v1/consignments/{id}
func (m *Manager) HandleGetConsignmentByID(w http.ResponseWriter, r *http.Request) {
	m.consignmentRouter.HandleGetConsignmentByID(w, r)
}

// pluginStateToWorkflowNodeState converts a plugin.State to a WorkflowNodeState.
// Returns an error if the plugin state is not recognized.
func pluginStateToWorkflowNodeState(state plugin.State) (model.WorkflowNodeState, error) {
	switch state {
	case plugin.InProgress:
		return model.WorkflowNodeStateInProgress, nil
	case plugin.Completed:
		return model.WorkflowNodeStateCompleted, nil
	case plugin.Failed:
		return model.WorkflowNodeStateFailed, nil
	default:
		return "", fmt.Errorf("unknown plugin state: %s", state)
	}
}
