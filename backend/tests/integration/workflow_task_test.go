package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/workflow"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// setupTestEnv initializes a clean in-memory environment for each test
func setupTestEnv(t *testing.T) (taskManager.TaskManager, *gorm.DB, chan taskManager.WorkflowManagerNotification) {
	// Unique DB name per test run to avoid shared state in cache=shared
	dbName := uuid.New().String()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate
	// Note: We need to migrate all models involved. 
	// Adjust list based on actual models in the codebase.
	// Assuming these models exist and are correct based on usage in services.
	err = db.AutoMigrate(
		&model.Consignment{},
		&model.HSCode{},
		&model.WorkflowTemplate{},
		&model.WorkflowNodeTemplate{},
		&model.WorkflowNode{},
		&model.WorkflowTemplateMap{},
	)
	if err != nil {
		// Try to migrate minimal set if full set fails, or just log
		t.Logf("AutoMigrate warning (might be missing models in test scope): %v", err)
	}
	
	// Task Persistence Models? 
	// The TaskManager creates its own store which migrates models?
	// persistence.NewTaskStore(db) usually calls AutoMigrate.

	// Seed HS Code
	hsCode := model.HSCode{HSCode: "1234.56", Description: "Test Item"}
	if err := db.Create(&hsCode).Error; err != nil {
		t.Fatalf("failed to seed HS code: %v", err)
	}

	// Seed Workflow Template (New Architecture uses Nodes)
	// We need 2 simple form nodes.
	node1ID := uuid.New()
	node2ID := uuid.New()

	nodeTemplate1 := model.WorkflowNodeTemplate{
		BaseModel: model.BaseModel{ID: node1ID},
		Name:      "Step 1",
		Type:      plugin.TaskTypeSimpleForm, // "SIMPLE_FORM"
		Config:    json.RawMessage(`{"formId":"f1"}`),
		DependsOn: []uuid.UUID{},
	}
	nodeTemplate2 := model.WorkflowNodeTemplate{
		BaseModel: model.BaseModel{ID: node2ID},
		Name:      "Step 2",
		Type:      plugin.TaskTypeSimpleForm,
		Config:    json.RawMessage(`{"formId":"f2"}`),
		DependsOn: []uuid.UUID{node1ID},
	}

	if err := db.Create(&nodeTemplate1).Error; err != nil {
		t.Fatalf("failed to create node 1: %v", err)
	}
	if err := db.Create(&nodeTemplate2).Error; err != nil {
		t.Fatalf("failed to create node 2: %v", err)
	}

	workflowTemplate := model.WorkflowTemplate{
		BaseModel:     model.BaseModel{ID: uuid.New()},
		Version:       "1.0",
		NodeTemplates: []uuid.UUID{node1ID, node2ID},
	}
	// Need to make sure Nodes are stored correctly. 
	// The model definition for WorkflowTemplate might store IDs as JSONArray or association.
	// Assuming standard JSON support for array of UUIDs or similar.
	
	if err := db.Create(&workflowTemplate).Error; err != nil {
		t.Fatalf("failed to create workflow template: %v", err)
	}

	// Map HS Code to Workflow
	mapping := model.WorkflowTemplateMap{
		HSCodeID:           hsCode.ID,
		ConsignmentFlow:    model.ConsignmentFlowImport,
		WorkflowTemplateID: workflowTemplate.ID,
	}

	if err := db.Create(&mapping).Error; err != nil {
		t.Fatalf("failed to create mapping: %v", err)
	}

	taskUpdateChan := make(chan taskManager.WorkflowManagerNotification, 100)
	
	// Initialize Form Service
	formService := form.NewFormService(db)

	cfg := &config.Config{} // Empty config or mock if needed

	// Initialize Task Manager
	tm, err := taskManager.NewTaskManager(db, taskUpdateChan, cfg, formService)
	if err != nil {
		t.Fatalf("failed to create task manager: %v", err)
	}

	// Re-initialize Consignment Service via NewManager to make sure everything used in test logic is consistent
	// But we can just return the one from manager or create separate. 
	// The test needs `wm` to handle http requests.
	
	// However, setupTestEnv returns `service.ConsignmentService`. 
	// We can get it from the logical flow. 
	// Actually, better to just return the components we use.
	
	// Create minimal services for the return signature
	// But actually, we should just let the caller create the WorkflowManager with these deps.
	
	// We will create a fresh ConsignmentService just for the return signature if needed,
	// but mostly we should use the one inside WorkflowManager or the API.
	
	// Initialize minimal services for "cs" return
	// ts := service.NewTemplateService(db)
	// wns := service.NewWorkflowNodeService(db)
	// cs := service.NewConsignmentService(db, ts, wns)

	return tm, db, taskUpdateChan
}

func TestWorkflowFlows(t *testing.T) {
	// Skip if running in short mode or if no suitable environment
	// if testing.Short() { t.Skip("skipping integration test") }

	t.Run("Success Path - Full Completion", func(t *testing.T) {
		tm, db, updateChan := setupTestEnv(t)
		wm := workflow.NewManager(tm, updateChan, db)

		// 1. Create Consignment
		// traderID := "trader-1"
		req := &model.CreateConsignmentDTO{
			// TraderID:  &traderID, // Removed as it's not in DTO
			Flow:  model.ConsignmentFlowImport,
			Items: []model.CreateConsignmentItemDTO{{HSCodeID: getHSCodeID(t, db)}},
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/consignments", bytes.NewReader(body))
		w := httptest.NewRecorder()
		wm.HandleCreateConsignment(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create consignment: %d %s", w.Code, w.Body.String())
		}

		var consignment model.ConsignmentResponseDTO
		if err := json.NewDecoder(w.Body).Decode(&consignment); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		t.Logf("Consignment created: %s", consignment.ID)

		// 2. Poll for Step 1 Task to be READY
		// The workflow manager runs a background goroutine that registers tasks.
		// We need to wait for it to happen.
		step1Node, err := waitForNodeStatus(t, db, consignment.ID, "Step 1", model.WorkflowNodeStateInProgress) 
		// Note: Initial state after InitTask is InProgress because InitTask calls start() which notifies InProgress?
		// Or maybe READY? 
		// Logic in Manager.registerWorkflowNodesWithTaskManager -> InitTask -> start -> notifies InProgress?
		// Usually SimpleForm task starts in InProgress (waiting for user input).
		if err != nil {
			t.Fatalf("Failed waiting for Step 1: %v", err)
		}

		// 3. Complete Step 1
		t.Logf("Executing Step 1 Task: %s", step1Node.ID)
		
		executeTaskSync(t, tm, consignment.ID, step1Node.ID, "SUBMISSION")

		// 4. Poll for Step 1 to be COMPLETED and Step 2 to be IN_PROGRESS
		// Since we have dependencies (Step 2 depends on Step 1), 
		// logic: Step 1 Complete -> Workflow Listener -> Update Step 1 -> Check Step 2 -> Step 2 Ready -> Register Step 2 -> Step 2 InProgress
		
		step2Node, err := waitForNodeStatus(t, db, consignment.ID, "Step 2", model.WorkflowNodeStateInProgress)
		if err != nil {
			t.Fatalf("Failed waiting for Step 2: %v", err)
		}

		// 5. Complete Step 2
		t.Logf("Executing Step 2 Task: %s", step2Node.ID)
		executeTaskSync(t, tm, consignment.ID, step2Node.ID, "SUBMISSION")

		// 6. Poll for Consignment to be COMPLETED
		if err := waitForConsignmentState(t, db, consignment.ID, model.ConsignmentStateFinished); err != nil {
			t.Fatalf("Consignment did not finish: %v", err)
		}
		
		// Also verify Step 2 is COMPLETED
		var checkNode2 model.WorkflowNode
		db.First(&checkNode2, "id = ?", step2Node.ID)
		if checkNode2.State != model.WorkflowNodeStateCompleted {
			t.Errorf("Expected Step 2 to be COMPLETED, got %s", checkNode2.State)
		}
	})
	
	// Failure path can be added similarly
}

// Helpers

func executeTaskSync(t *testing.T, tm taskManager.TaskManager, wID, tID uuid.UUID, action string) {
	// Build payload
	payload := &plugin.ExecutionRequest{
		Action:  action,
		Content: map[string]interface{}{"test": "data"},
	}
	
	reqBody := taskManager.ExecuteTaskRequest{
		WorkflowID: wID,
		TaskID:     tID,
		Payload:    payload,
	}
	
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	tm.HandleExecuteTask(w, req)
	
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteTask failed: %d %s", w.Code, w.Body.String())
	}
	
	// Verify success in response
	var resp taskManager.ExecuteTaskResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("ExecuteTask returned failure: %s", resp.Error)
	}
}

func getHSCodeID(t *testing.T, db *gorm.DB) uuid.UUID {
	var hs model.HSCode
	if err := db.First(&hs).Error; err != nil {
		t.Fatalf("Found no HS Code: %v", err)
	}
	return hs.ID
}

// waitForNodeStatus polls the DB until a node with the given name (via template) has the expected state
func waitForNodeStatus(t *testing.T, db *gorm.DB, consignmentID uuid.UUID, nodeName string, expectedState model.WorkflowNodeState) (*model.WorkflowNode, error) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for node '%s' to be %s", nodeName, expectedState)
		case <-ticker.C:
			// Find the node
			// We need to join with Template to check name
			
			// This query assumes we can join or we just fetch all and check.
			// Fetching all nodes for consignment
			var nodes []model.WorkflowNode
			db.Where("consignment_id = ?", consignmentID).Find(&nodes)
			
			for _, n := range nodes {
				// Get template to check name (inefficient but fine for test)
				var tpl model.WorkflowNodeTemplate
				if err := db.First(&tpl, "id = ?", n.WorkflowNodeTemplateID).Error; err == nil {
					if tpl.Name == nodeName {
						if n.State == expectedState {
							match := n
							return &match, nil
						}
					}
				}
			}
		}
	}
}

func waitForConsignmentState(t *testing.T, db *gorm.DB, consignmentID uuid.UUID, expectedState model.ConsignmentState) error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for consignment to be %s", expectedState)
		case <-ticker.C:
			var c model.Consignment
			if err := db.First(&c, "id = ?", consignmentID).Error; err != nil {
				continue
			}
			if c.State == expectedState {
				return nil
			}
		}
	}
}
