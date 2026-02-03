package workflow

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/OpenNSW/nsw/internal/workflow/r_router"
	"github.com/OpenNSW/nsw/internal/workflow/r_service"
	"gorm.io/gorm"
)

// R_Manager is the refactored workflow manager that coordinates between services, routers, and task manager
type R_Manager struct {
	tm                     task.TaskManager
	hsCodeService          *r_service.HSCodeService
	consignmentService     *r_service.ConsignmentService
	workflowNodeService    *r_service.WorkflowNodeService
	hsCodeRouter           *r_router.HSCodeRouter
	consignmentRouter      *r_router.ConsignmentRouter
	workflowNodeUpdateChan chan r_model.UpdateWorkflowNodeDTO
	ctx                    context.Context
	cancel                 context.CancelFunc
}

// NewR_Manager creates a new refactored workflow manager
func NewR_Manager(tm task.TaskManager, db *gorm.DB) *R_Manager {
	// Initialize services
	hsCodeService := r_service.NewHSCodeService(db)
	consignmentService := r_service.NewConsignmentService(db)
	workflowNodeService := r_service.NewWorkflowNodeService(db)

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	m := &R_Manager{
		tm:                     tm,
		hsCodeService:          hsCodeService,
		consignmentService:     consignmentService,
		workflowNodeService:    workflowNodeService,
		workflowNodeUpdateChan: make(chan r_model.UpdateWorkflowNodeDTO, 100),
		ctx:                    ctx,
		cancel:                 cancel,
	}

	// Initialize routers
	m.hsCodeRouter = r_router.NewHSCodeRouter(hsCodeService)
	m.consignmentRouter = r_router.NewConsignmentRouter(consignmentService, m.registerWorkflowNodes)

	return m
}

// StartWorkflowNodeUpdateListener starts a goroutine that listens for workflow node updates
func (m *R_Manager) StartWorkflowNodeUpdateListener() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				slog.Info("workflow node update listener stopped")
				return
			case update := <-m.workflowNodeUpdateChan:
				// TODO: Implement workflow node update propagation logic
				// This would handle state transitions and dependency unlocking
				slog.Info("workflow node update received", "update", update)
			}
		}
	}()
}

// StopWorkflowNodeUpdateListener stops the workflow node update listener
func (m *R_Manager) StopWorkflowNodeUpdateListener() {
	if m.cancel != nil {
		m.cancel()
	}
}

// registerWorkflowNodes registers workflow nodes with the Task Manager
// This is called when new READY workflow nodes are created
func (m *R_Manager) registerWorkflowNodes(workflowNodes []r_model.WorkflowNode, consignmentGlobalContext map[string]interface{}) {
	for _, node := range workflowNodes {
		// TODO: Map WorkflowNode to Task structure when task refactoring is complete
		// For now, this is a placeholder for the integration point
		slog.Info("registering workflow node",
			"nodeID", node.ID,
			"consignmentID", node.ConsignmentID,
			"state", node.State,
		)

		// Future implementation:
		// initPayload := task.InitPayload{
		// 	TaskID:        node.ID,
		// 	Type:          mapNodeTypeToTaskType(node.Type),
		// 	Status:        mapNodeStateToTaskStatus(node.State),
		// 	ConsignmentID: node.ConsignmentID,
		// 	GlobalContext: consignmentGlobalContext,
		// }
		// _, err := m.tm.RegisterTask(context.Background(), initPayload)
		// if err != nil {
		// 	slog.Error("failed to register workflow node as task", "nodeID", node.ID, "error", err)
		// }
	}
}

// HTTP Handler delegation methods

// HandleGetAllHSCodes handles GET /api/v1/hscodes
func (m *R_Manager) HandleGetAllHSCodes(w http.ResponseWriter, r *http.Request) {
	m.hsCodeRouter.HandleGetAllHSCodes(w, r)
}

// HandleCreateConsignment handles POST /api/v1/consignments
func (m *R_Manager) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	m.consignmentRouter.HandleCreateConsignment(w, r)
}
