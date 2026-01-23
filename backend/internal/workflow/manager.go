package workflow

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"gorm.io/gorm"
)

type Manager struct {
	tm             task.TaskManager
	cs             *service.ConsignmentService
	wr             *router.WorkflowRouter
	taskUpdateChan chan model.TaskCompletionNotification
}

func NewManager(tm task.TaskManager, taskUpdateChan chan model.TaskCompletionNotification, db *gorm.DB) *Manager {
	ts := service.NewTaskService(db)
	cs := service.NewConsignmentService(ts, db)

	m := &Manager{
		tm:             tm,
		cs:             cs,
		taskUpdateChan: taskUpdateChan,
	}

	// Create router with callback to register tasks
	m.wr = router.NewWorkflowRouter(cs, m.registerTasks)

	return m
}

// StartTaskUpdateListener starts a goroutine that listens for task completion notifications
func (m *Manager) StartTaskUpdateListener() {
	go func() {
		for update := range m.taskUpdateChan {
			newReadyTasks, _ := m.cs.UpdateTaskStatusAndPropagateChanges(
				context.Background(),
				update.TaskID,
				update.State,
			)
			// Register newly ready tasks with Task Manager
			if len(newReadyTasks) > 0 {
				m.registerTasks(newReadyTasks)
			}
		}
	}()
}

// registerTasks registers multiple tasks with Task Manager
func (m *Manager) registerTasks(tasks []*model.Task) {
	for _, t := range tasks {
		initPayload := task.InitPayload{
			TaskID:        t.ID,
			Type:          task.Type(t.Type),
			Status:        t.Status,
			CommandSet:    t.Config,
			ConsignmentID: t.ConsignmentID,
			StepID:        t.StepID,
		}
		_, err := m.tm.RegisterTask(context.Background(), initPayload)
		if err != nil {
			slog.Error("failed to register task", "taskID", t.ID, "error", err)
			return
		}
	}
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
