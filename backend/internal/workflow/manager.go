package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	tm             task.TaskManager
	cs             *service.ConsignmentService
	wr             *router.WorkflowRouter
	taskUpdateChan chan model.TaskCompletionNotification
	formService    form.FormService
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewManager(tm task.TaskManager, taskUpdateChan chan model.TaskCompletionNotification, db *gorm.DB, formService form.FormService) *Manager {
	ts := service.NewTaskService(db)
	cs := service.NewConsignmentService(ts, db)

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		tm:             tm,
		cs:             cs,
		taskUpdateChan: taskUpdateChan,
		formService:    formService,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Set up task fetcher callback for Task Manager to fetch tasks from workflow database
	tm.SetTaskFetcher(func(ctx context.Context, taskID uuid.UUID) (*task.InitPayload, error) {
		workflowTask, err := ts.GetTaskByID(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch task from workflow database: %w", err)
		}

		return &task.InitPayload{
			TaskID:        workflowTask.ID,
			Type:          task.Type(workflowTask.Type),
			Status:        workflowTask.Status,
			CommandSet:    workflowTask.Config,
			ConsignmentID: workflowTask.ConsignmentID,
			StepID:        workflowTask.StepID,
		}, nil
	})

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
// Note: OGA service now polls GET /api/tasks to discover OGA_FORM tasks, so no notification is needed
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

	// For OGA_FORM tasks, return embedded form schema based on agency/service
	if record.Type == "OGA_FORM" {
		var ogaConfig struct {
			Agency  string `json:"agency"`
			Service string `json:"service"`
		}
		if err := json.Unmarshal(record.CommandSet, &ogaConfig); err == nil && ogaConfig.Agency != "" {
			// Return embedded OGA review form
			ogaForm := map[string]interface{}{
				"id":      record.ID,
				"name":    ogaConfig.Agency + " Review Form",
				"version": "1.0",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"inspectionDate": map[string]interface{}{
							"type":  "string",
							"title": "Inspection Date",
						},
						"certificateNumber": map[string]interface{}{
							"type":  "string",
							"title": "Certificate Number",
						},
						"inspectorName": map[string]interface{}{
							"type":  "string",
							"title": "Inspector Name",
						},
						"remarks": map[string]interface{}{
							"type":  "string",
							"title": "Remarks",
						},
					},
				},
				"uiSchema": map[string]interface{}{
					"remarks": map[string]interface{}{
						"ui:widget": "textarea",
					},
				},
			}
			m.writeJSONResponse(w, http.StatusOK, ogaForm)
			return
		}
	}

	// For TRADER_FORM tasks, look up form by formId
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

	ctx := r.Context()

	// Get OGA task to find consignment ID and step ID
	ogaTask, err := m.tm.GetTask(taskID)
	if err != nil {
		m.writeJSONError(w, http.StatusNotFound, "OGA task not found")
		return
	}

	// Get consignment from workflow database to access steps with dependencies
	consignment, err := m.cs.GetConsignmentByID(ctx, ogaTask.ConsignmentID)
	if err != nil {
		m.writeJSONError(w, http.StatusNotFound, "consignment not found")
		return
	}

	// Find the OGA_FORM step that matches this taskId
	var ogaStep *model.ConsignmentStep
	var ogaStepFound bool
	for _, item := range consignment.Items {
		for i := range item.Steps {
			if item.Steps[i].TaskID == taskID && item.Steps[i].Type == "OGA_FORM" {
				ogaStep = &item.Steps[i]
				ogaStepFound = true
				break
			}
		}
		if ogaStepFound {
			break
		}
	}

	if !ogaStepFound || ogaStep == nil {
		m.writeJSONError(w, http.StatusNotFound, "OGA step not found in consignment")
		return
	}

	// Find TRADER_FORM step(s) that are dependencies of this OGA_FORM step
	var traderTaskIDs []uuid.UUID
	for _, item := range consignment.Items {
		for _, step := range item.Steps {
			// Check if this step is a dependency of the OGA_FORM step
			for _, depStepID := range ogaStep.DependsOn {
				if step.StepID == depStepID && step.Type == "TRADER_FORM" && step.Status == "COMPLETED" {
					traderTaskIDs = append(traderTaskIDs, step.TaskID)
				}
			}
		}
	}

	// If no trader task found in dependencies, return placeholder
	if len(traderTaskIDs) == 0 {
		m.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Trader has not yet submitted the required form",
			"status":  "PENDING",
		})
		return
	}

	// Fetch trader submission from the first TRADER_FORM task (or merge multiple if needed)
	traderTaskID := traderTaskIDs[0]
	traderTask, err := m.tm.GetTask(traderTaskID)
	if err != nil {
		m.writeJSONError(w, http.StatusNotFound, "trader task not found")
		return
	}

	var resultData interface{}
	if len(traderTask.ResultData) > 0 {
		if err := json.Unmarshal(traderTask.ResultData, &resultData); err != nil {
			m.writeJSONError(w, http.StatusInternalServerError, "failed to parse trader submission")
			return
		}
	} else {
		// Return empty object if no result data
		resultData = map[string]interface{}{}
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
