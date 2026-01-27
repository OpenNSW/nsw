package workflow

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/OpenNSW/nsw/oga"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	tm             task.TaskManager
	cs             *service.ConsignmentService
	wr             *router.WorkflowRouter
	taskUpdateChan chan model.TaskCompletionNotification
	ogaClient      oga.Client
	formService    form.FormService
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewManager(tm task.TaskManager, taskUpdateChan chan model.TaskCompletionNotification, db *gorm.DB, ogaClient oga.Client, formService form.FormService) *Manager {
	ts := service.NewTaskService(db)
	cs := service.NewConsignmentService(ts, db)

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		tm:             tm,
		cs:             cs,
		taskUpdateChan: taskUpdateChan,
		ogaClient:      ogaClient,
		formService:    formService,
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

		// Notify OGA Service if this is an OGA_FORM task that became IN_PROGRESS
		// This logic is placed here to keep TaskManager decoupled from OGA Service dependency
		if t.Type == "OGA_FORM" && t.Status == "IN_PROGRESS" && m.ogaClient != nil {
			var config struct {
				FormID string `json:"formId"`
			}
			// Best effort to parse config and get formID
			if err := json.Unmarshal(t.Config, &config); err != nil {
				slog.Warn("failed to parse task config for OGA notification", "taskID", t.ID, "error", err)
				// Continue, maybe formID is optional or we can send without it? 
				// The OGA service expects formID, so maybe we skip or send empty? 
				// Let's log and proceed.
			}

			notification := oga.OGATaskNotification{
				TaskID:        t.ID,
				ConsignmentID: t.ConsignmentID,
				FormID:        config.FormID,
				Status:        string(t.Status),
			}

			go func() {
				if err := m.ogaClient.NotifyApplicationReady(context.Background(), notification); err != nil {
					slog.Warn("failed to notify OGA service", "taskID", t.ID, "error", err)
				} else {
					slog.Info("notified OGA service of new application", "taskID", t.ID)
				}
			}()
		}
	}
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

// HandleGetTasks handles GET requests to retrieve tasks with optional filters
func (m *Manager) HandleGetTasks(w http.ResponseWriter, r *http.Request) {
	m.wr.HandleGetTasks(w, r)
}

// HandleGetTaskForm returns the form schema for a task
func (m *Manager) HandleGetTaskForm(w http.ResponseWriter, r *http.Request) {
	taskIDStr := r.PathValue("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		m.writeJSONError(w, http.StatusBadRequest, "invalid taskId")
		return
	}

	record, err := m.tm.GetTask(taskID)
	if err != nil {
		m.writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}

	var config struct {
		FormID uuid.UUID `json:"formId"`
	}
	if err := json.Unmarshal(record.CommandSet, &config); err != nil {
		m.writeJSONError(w, http.StatusInternalServerError, "failed to parse task config")
		return
	}

	if config.FormID == uuid.Nil {
		m.writeJSONError(w, http.StatusBadRequest, "task config does not contain formId")
		return
	}

	if m.formService == nil {
		m.writeJSONError(w, http.StatusInternalServerError, "form service not configured")
		return
	}

	formResp, err := m.formService.GetFormByID(r.Context(), config.FormID)
	if err != nil {
		m.writeJSONError(w, http.StatusInternalServerError, "failed to retrieve form schema: "+err.Error())
		return
	}

	m.writeJSONResponse(w, http.StatusOK, formResp)
}

// HandleGetTraderSubmission returns the data submitted by trader for a consignment
func (m *Manager) HandleGetTraderSubmission(w http.ResponseWriter, r *http.Request) {
	taskIDStr := r.PathValue("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		m.writeJSONError(w, http.StatusBadRequest, "invalid taskId")
		return
	}

	ogaTask, err := m.tm.GetTask(taskID)
	if err != nil {
		m.writeJSONError(w, http.StatusNotFound, "OGA task not found")
		return
	}

	// Get all tasks for this consignment
	tasks, err := m.tm.GetTasksByConsignment(ogaTask.ConsignmentID)
	if err != nil {
		m.writeJSONError(w, http.StatusInternalServerError, "failed to retrieve tasks for consignment")
		return
	}

	// Find TRADER_FORM task
	var traderTask *task.TaskRecord
	for i, t := range tasks {
		if t.Type == "TRADER_FORM" {
			traderTask = &tasks[i]
			break
		}
	}

	if traderTask == nil {
		m.writeJSONError(w, http.StatusNotFound, "trader submission not found")
		return
	}

	var resultData interface{}
	if len(traderTask.ResultData) > 0 {
		if err := json.Unmarshal(traderTask.ResultData, &resultData); err != nil {
			m.writeJSONError(w, http.StatusInternalServerError, "failed to parse trader submission")
			return
		}
	}

	m.writeJSONResponse(w, http.StatusOK, resultData)
}

func (m *Manager) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func (m *Manager) writeJSONError(w http.ResponseWriter, status int, message string) {
	m.writeJSONResponse(w, status, map[string]string{"error": message})
}
