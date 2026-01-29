package workflow

import (
	"context"
	"encoding/json"
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
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewManager(tm task.TaskManager, taskUpdateChan chan model.TaskCompletionNotification, db *gorm.DB) *Manager {
	ts := service.NewTaskService(db)
	cs := service.NewConsignmentService(ts, db)

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		tm:             tm,
		cs:             cs,
		taskUpdateChan: taskUpdateChan,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Create router with callback to register tasks
	m.wr = router.NewWorkflowRouter(cs, m.registerTasks)

	return m
}

// StartTaskUpdateListener starts a goroutine that listens for task completion notifications
func (m *Manager) StartTaskUpdateListener() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				slog.Info("task update listener stopped")
				return
			case update := <-m.taskUpdateChan:
				newReadyTasks, consignment, err := m.cs.UpdateTaskStatusAndPropagateChanges(
					context.Background(),
					update.TaskID,
					update.State,
					update.AppendGlobalContext,
				)
				if err != nil {
					slog.Error("failed to process task completion notification", "taskID", update.TaskID, "error", err)
					continue
				}

				// Log if consignment is completed
				if consignment != nil && consignment.State == model.ConsignmentStateFinished {
					slog.Info("consignment finished", "consignmentID", consignment.ID)
				}
				// Register newly ready tasks with Task Manager
				if len(newReadyTasks) > 0 {
					m.registerTasks(newReadyTasks, consignment.GlobalContext)
				}
			}
		}
	}()
}

// StopTaskUpdateListener stops the task update listener by canceling the context
func (m *Manager) StopTaskUpdateListener() {
	if m.cancel != nil {
		m.cancel()
	}
}

// registerTasks registers multiple tasks with Task Manager
func (m *Manager) registerTasks(tasks []*model.Task, consignmentGlobalContext map[string]interface{}) {
	for _, t := range tasks {
		initPayload := task.InitPayload{
			TaskID:        t.ID,
			Type:          string(t.Type),
			Status:        string(t.Status),
			Config:        t.Config,
			ConsignmentID: t.ConsignmentID,
			StepID:        t.StepID,
			GlobalContext: consignmentGlobalContext,
		}
		container, err := m.tm.InitTaskContainer(context.Background(), initPayload)
		if err != nil {
			slog.Error("failed to init task container", "taskID", t.ID, "error", err)
			continue
		}


		// Prepare config map
		// We need to unmarshal the container config and merge it with global context identifiers
		var configMap map[string]any
		if len(t.Config) > 0 {
			if err := json.Unmarshal(t.Config, &configMap); err != nil {
				slog.Error("failed to unmarshal task config, skipping task start", "taskID", t.ID, "error", err)
				continue
			}
		} else {
			configMap = make(map[string]any)
		}
		
		// Add context IDs to config
		configMap["taskId"] = t.ID.String()
		configMap["consignmentId"] = t.ConsignmentID.String()

		// Start the task via Supervisor Execute
		status, _, err := container.Execute(context.Background(), configMap)
		if err != nil {
			slog.Error("failed to start task", "taskID", t.ID, "error", err)
			continue
		}

		// Notify state if changed
		if status != task.TaskStatusAwaitingInput {
			m.tm.NotifyState(context.Background(), t.ID, status, container.GlobalState.GetAll())
		}
	}
}

// HandleGetHSCode handles GET requests for a specific HS code by ID
func (m *Manager) HandleGetHSCodeID(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleGetHSCodeID(w, r)
}

// HandleGetHSCodes handles GET requests for HS codes
func (m *Manager) HandleGetHSCodes(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleGetHSCodes(w, r)
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

// HandleGetConsignments handles GET requests to retrieve consignments
func (m *Manager) HandleGetConsignments(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleGetConsignments(w, r)
}
