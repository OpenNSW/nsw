package workflow

import (
	"context"
	"log/slog"
	"net/http"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/OpenNSW/nsw/internal/workflow/r_router"
	"github.com/OpenNSW/nsw/internal/workflow/r_service"
	"gorm.io/gorm"
)

// Manager is the refactored workflow manager that coordinates between services, routers, and task manager
type Manager struct {
	tm                     taskManager.TaskManager
	hsCodeService          *r_service.HSCodeService
	consignmentService     *r_service.ConsignmentService
	workflowNodeService    *r_service.WorkflowNodeService
	templateService        *r_service.TemplateService
	hsCodeRouter           *r_router.HSCodeRouter
	consignmentRouter      *r_router.ConsignmentRouter
	workflowNodeUpdateChan chan taskManager.WorkflowManagerNotification
	ctx                    context.Context
	cancel                 context.CancelFunc
}

// NewManager creates a new refactored workflow manager
func NewManager(tm taskManager.TaskManager, ch chan taskManager.WorkflowManagerNotification, db *gorm.DB) *Manager {
	// Initialize services
	hsCodeService := r_service.NewHSCodeService(db)
	consignmentService := r_service.NewConsignmentService(db)
	workflowNodeService := r_service.NewWorkflowNodeService(db)
	templateService := r_service.NewTemplateService(db)

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

	// Initialize routers
	m.hsCodeRouter = r_router.NewHSCodeRouter(hsCodeService)
	m.consignmentRouter = r_router.NewConsignmentRouter(consignmentService, m.registerWorkflowNodesWithTaskManager)

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
				updateReq := r_model.UpdateWorkflowNodeDTO{
					WorkflowNodeID:      update.TaskID,
					State:               r_model.WorkflowNodeState(*update.UpdatedState),
					AppendGlobalContext: update.AppendGlobalContext,
				}
				newReadyNodes, newGlobalContext, err := m.consignmentService.UpdateWorkflowNodeStateAndPropagateChanges(m.ctx, &updateReq)
				if err != nil {
					slog.Error("failed to handle workflow node update", "error", err)
					continue
				}
				if len(newReadyNodes) > 0 {
					m.registerWorkflowNodesWithTaskManager(newReadyNodes, newGlobalContext)
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
func (m *Manager) registerWorkflowNodesWithTaskManager(workflowNodes []r_model.WorkflowNode, consignmentGlobalContext map[string]any) {
	for _, node := range workflowNodes {
		nodeTemplate, err := m.templateService.GetWorkflowNodeTemplateByID(m.ctx, node.WorkflowNodeTemplateID)
		if err != nil {
			slog.Error("failed to get workflow node template", "error", err)
			continue
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
			slog.Error("failed to initialize task in task manager", "error", err)
			continue
		}
		slog.Info("successfully registered workflow node with task manager", "Response", response.Result)
	}

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
