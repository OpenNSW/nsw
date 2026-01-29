package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/task"
	"github.com/OpenNSW/nsw/internal/workflow"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestEnv initializes a clean in-memory environment for each test
func setupTestEnv(t *testing.T) (*service.ConsignmentService, task.TaskManager, *gorm.DB, chan model.TaskCompletionNotification) {
	// Unique DB name per test run to avoid shared state in cache=shared
	dbName := uuid.New().String()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate
	db.AutoMigrate(&model.Consignment{}, &model.HSCode{}, &model.WorkflowTemplate{}, &model.WorkflowTemplateMap{}, &model.Task{})

	// Seed HS Code and Template
	hsCode := model.HSCode{HSCode: "1234.56", Description: "Test Item"}
	db.Create(&hsCode)

	template := model.WorkflowTemplate{
		Version: "1.0",
		Steps: []model.Step{
			{StepID: "step1", Type: model.StepTypeSimpleForm, Config: []byte(`{"formId":"cusdec_declaration"}`)},
			{StepID: "step2", Type: model.StepTypeSimpleForm, Config: []byte(`{"formId":"phytosanitary_cert"}`), DependsOn: []string{"step1"}},
		},
	}
	db.Create(&template)

	db.Create(&model.WorkflowTemplateMap{
		HSCodeID:           hsCode.ID,
		TradeFlow:          model.TradeFlowImport,
		WorkflowTemplateID: template.ID,
	})

	taskUpdateChan := make(chan model.TaskCompletionNotification, 10)
	tm, err := task.NewTaskManager(fmt.Sprintf("file:%s_tasks?mode=memory&cache=shared", dbName), taskUpdateChan, &config.Config{})
	if err != nil {
		t.Fatalf("failed to create task manager: %v", err)
	}
	ts := service.NewTaskService(db)
	cs := service.NewConsignmentService(ts, db)

	return cs, tm, db, taskUpdateChan
}

func TestWorkflowFlows(t *testing.T) {
	ctx := context.Background()

	t.Run("Success Path - Full Completion", func(t *testing.T) {
		cs, tm, db, updateChan := setupTestEnv(t)
		wm := workflow.NewManager(tm, updateChan, db)

		// 1. Create Consignment
		traderID := "trader-1"
		req := &model.CreateConsignmentDTO{
			TraderID:  &traderID,
			TradeFlow: model.TradeFlowImport,
			Items:     []model.CreateWorkflowForItemDTO{{HSCodeID: getHSCodeID(t, db)}},
		}
		
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/consignments", bytes.NewReader(body))
		w := httptest.NewRecorder()
		wm.HandleCreateConsignment(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create consignment: %v", w.Body.String())
		}

		var consignment model.ConsignmentResponse
		json.NewDecoder(w.Body).Decode(&consignment)

		// 2. Complete Step 1
		step1 := consignment.Items[0].Steps[0]
		executeTaskSync(t, tm, consignment.ID, step1.TaskID, "SUBMIT_FORM")

		// 3. Manually process notification
		update := <-updateChan
		newTasks, updatedC, err := cs.UpdateTaskStatusAndPropagateChanges(ctx, update.TaskID, update.State, update.AppendGlobalContext)
		if err != nil {
			t.Fatalf("UpdateTaskStatusAndPropagateChanges failed: %v", err)
		}

		// 4. Register newly ready tasks
		for _, t := range newTasks {
			tm.RegisterTask(ctx, task.InitPayload{
				TaskID:        t.ID,
				Type:          task.Type(t.Type),
				Status:        t.Status,
				CommandSet:    t.Config,
				ConsignmentID: t.ConsignmentID,
				StepID:        t.StepID,
				GlobalContext: updatedC.GlobalContext,
			})
		}

		// 5. Verify Step 2 is now READY
		var c model.Consignment
		db.First(&c, "id = ?", consignment.ID)
		if c.Items[0].Steps[1].Status != model.TaskStatusReady {
			t.Errorf("Expected step2 to be READY, got %s", c.Items[0].Steps[1].Status)
		}

		// 6. Complete Step 2
		executeTaskSync(t, tm, consignment.ID, c.Items[0].Steps[1].TaskID, "SUBMIT_FORM")
		update = <-updateChan
		_, finalC, err := cs.UpdateTaskStatusAndPropagateChanges(ctx, update.TaskID, update.State, update.AppendGlobalContext)
		if err != nil {
			t.Fatalf("UpdateTaskStatusAndPropagateChanges failed: %v", err)
		}

		// 7. Verify Consignment is FINISHED
		if finalC.State != model.ConsignmentStateFinished {
			t.Errorf("Expected consignment to be FINISHED, got %s", finalC.State)
		}
	})

	t.Run("Failure Path - Rejection", func(t *testing.T) {
		cs, tm, db, updateChan := setupTestEnv(t)
		wm := workflow.NewManager(tm, updateChan, db)

		traderID := "trader-2"
		req := &model.CreateConsignmentDTO{
			TraderID:  &traderID,
			TradeFlow: model.TradeFlowImport,
			Items:     []model.CreateWorkflowForItemDTO{{HSCodeID: getHSCodeID(t, db)}},
		}
		
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/consignments", bytes.NewReader(body))
		w := httptest.NewRecorder()
		wm.HandleCreateConsignment(w, httpReq)

		var consignment model.ConsignmentResponse
		json.NewDecoder(w.Body).Decode(&consignment)

		// Reject Step 1
		step1 := consignment.Items[0].Steps[0]
		executeTaskSync(t, tm, consignment.ID, step1.TaskID, "REJECT_FORM")

		// Process notification
		update := <-updateChan
		_, finalC, err := cs.UpdateTaskStatusAndPropagateChanges(context.Background(), update.TaskID, update.State, update.AppendGlobalContext)
		if err != nil {
			t.Fatalf("UpdateTaskStatusAndPropagateChanges failed: %v", err)
		}

		// Verify state is REQUIRES_REWORK
		if finalC.State != model.ConsignmentStateRequiresRework {
			t.Errorf("Expected REQUIRES_REWORK, got %s", finalC.State)
		}
	})
}

// Helpers
func executeTaskSync(t *testing.T, tm task.TaskManager, cID, tID uuid.UUID, action string) {
	payload := task.ExecuteTaskRequest{
		ConsignmentID: cID,
		TaskID:        tID,
		Payload: &task.ExecutionPayload{
			Action:  action,
			Content: map[string]interface{}{"test": "data"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	tm.HandleExecuteTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteTask failed: %d %s", w.Code, w.Body.String())
	}
}

func getHSCodeID(t *testing.T, db *gorm.DB) uuid.UUID {
	var hs model.HSCode
	db.First(&hs)
	return hs.ID
}