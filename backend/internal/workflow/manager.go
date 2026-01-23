package workflow

import (
	"context"
	"net/http"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	cs             *service.ConsignmentService
	wr             *router.WorkflowRouter
	taskUpdateChan *chan model.TaskCompletionNotification
}

func NewManager(taskUpdateChan *chan model.TaskCompletionNotification, db *gorm.DB) *Manager {
	ts := service.NewTaskService(db)
	cs := service.NewConsignmentService(ts, db)
	wr := router.NewWorkflowRouter(cs)

	return &Manager{
		cs:             cs,
		wr:             wr,
		taskUpdateChan: taskUpdateChan,
	}
}

// StartTaskUpdateListener starts a goroutine that listens for task completion notifications
func (m *Manager) StartTaskUpdateListener() {
	go func() {
		for update := range *m.taskUpdateChan {
			newReadyTask, _ := m.cs.UpdateTaskStatusAndPropagateChanges(
				context.Background(),
				update.TaskID,
				update.State,
			)
			// TODO: newReadyTask need to be processed by Task Manager (not implemented yet)
			_ = newReadyTask
		}
	}()
}

func (m *Manager) GetWorkFlowTemplate(ctx context.Context, hscode string, consignmentType model.ConsignmentType) (*model.WorkflowTemplate, error) {
	return m.cs.GetWorkFlowTemplate(ctx, hscode, consignmentType)
}

func (m *Manager) InitializeConsignment(ctx context.Context, createReq *model.CreateConsignmentDTO) (*model.Consignment, error) {
	return m.cs.InitializeConsignment(ctx, createReq)
}

func (m *Manager) GetConsignmentByID(ctx context.Context, consignmentID uuid.UUID) (*model.Consignment, error) {
	return m.cs.GetConsignmentByID(ctx, consignmentID)
}

// HandleGetWorkflowTemplate handles GET requests for workflow templates
func (m *Manager) HandleGetWorkflowTemplate(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleGetWorkflowTemplate(w, r)
}

// HandleCreateConsignment handles POST requests to create a new consignment
func (m *Manager) HandleCreateConsignment(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleCreateConsignment(w, r)
}

// HandleGetConsignment handles GET requests to retrieve a consignment by ID
func (m *Manager) HandleGetConsignment(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleGetConsignment(w, r)
}
