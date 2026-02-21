package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func TestPreConsignmentService_InitializePreConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	// Service constructs StateMachine internally.
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	templateID := uuid.New()
	createReq := &model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}
	initialContext := map[string]any{"key": "value"}

	// Mocks for InitializePreConsignment
	// Get PreConsignmentTemplate
	workflowTemplateID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1 ORDER BY "pre_consignment_templates"."id" LIMIT \$2`).
		WithArgs(templateID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
			AddRow(templateID, workflowTemplateID, []byte("[]"))) // No dependencies

	// Get Workflow Template
	workflowTemplate := &model.WorkflowTemplate{
		BaseModel:     model.BaseModel{ID: workflowTemplateID},
		Name:          "Test WF Template",
		NodeTemplates: model.UUIDArray{},
	}
	mockTemplateProvider.On("GetWorkflowTemplateByID", ctx, workflowTemplateID).Return(workflowTemplate, nil)

	// Get Node Templates
	mockTemplateProvider.On("GetWorkflowNodeTemplatesByIDs", ctx, []uuid.UUID{}).Return([]model.WorkflowNodeTemplate{}, nil)

	// Begin Tx
	sqlMock.ExpectBegin()

	// Create PreConsignment (Insert)
	sqlMock.ExpectExec(`INSERT INTO "pre_consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Initialize Nodes (StateMachine) -> CreateWorkflowNodesInTx
	// Commit
	sqlMock.ExpectCommit()

	// Reload (Select with Preloads)
	pcID := uuid.New()
	// GORM First(&pc, "id = ?", pc.ID) generates WHERE id = $1 AND "pre_consignments"."id" = $2
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 AND "pre_consignments"."id" = \$2 ORDER BY "pre_consignments"."id" LIMIT \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "created_at", "updated_at", "pre_consignment_template_id"}).
			AddRow(pcID, traderID, "DRAFT", time.Now(), time.Now(), templateID))

	// Preload 1: PreConsignmentTemplate
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Test PC Template"))

	// Preload 2: WorkflowNodes
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."pre_consignment_id" = \$1`).
		WithArgs(pcID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id"}))

	resp, nodes, err := service.InitializePreConsignment(ctx, createReq, traderID, initialContext)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, nodes)
}

func TestPreConsignmentService_GetTraderPreConsignments(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	limit := 10
	offset := 0

	// Count Templates
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignment_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Find Templates
	templateID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" ORDER BY name ASC LIMIT \$1`).
		WithArgs(limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Test Template"))

	// Find PreConsignments for Trader
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
			AddRow(uuid.New(), traderID, "IN_PROGRESS", templateID))

	result, err := service.GetTraderPreConsignments(ctx, traderID, &offset, &limit)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Len(t, result.Items, 1)
}

func TestPreConsignmentService_GetPreConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	pcID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		// Expectation: First with Preloads
		// GORM: SELECT * FROM "pre_consignments" WHERE id = $1 ORDER BY "pre_consignments"."id" LIMIT $2
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state"}).
				AddRow(pcID, "trader1", "DRAFT"))

		// Preload 1: WorkflowNodes (Based on actual execution order)
		sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."pre_consignment_id" = \$1`).
			WithArgs(pcID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id"}).AddRow(uuid.New(), pcID))

		// Preload 2: WorkflowNodeTemplate (Nested)
		sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE "workflow_node_templates"."id" = \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uuid.New(), "Node Template"))

		// Preload 3: PreConsignmentTemplate
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uuid.New(), "Template"))

		resp, err := service.GetPreConsignmentByID(ctx, pcID)
		assert.NoError(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, pcID, resp.ID)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		resp, err := service.GetPreConsignmentByID(ctx, pcID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestPreConsignmentService_UpdateWorkflowNodeStateAndPropagateChanges(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	nodeID := uuid.New()
	pcID := uuid.New()

	t.Run("Success - Transition to InProgress", func(t *testing.T) {
		updateReq := &model.UpdateWorkflowNodeDTO{
			WorkflowNodeID: nodeID,
			State:          model.WorkflowNodeStateInProgress,
		}

		node := &model.WorkflowNode{
			BaseModel:        model.BaseModel{ID: nodeID},
			PreConsignmentID: &pcID,
			State:            model.WorkflowNodeStateReady,
		}

		sqlMock.ExpectBegin()

		// Get Node In Tx
		mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(node, nil).Once()

		// Update Node (StateMachine)
		mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 1 && nodes[0].State == model.WorkflowNodeStateInProgress
		})).Return(nil).Once()

		// Append Context (Success)
		// First(pre_consignment) for append context
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_context"}).AddRow(pcID, []byte("{}")))

		// Save(pre_consignment)
		sqlMock.ExpectExec(`UPDATE "pre_consignments"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		_, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
		assert.NoError(t, err)

		mockNodeRepo.AssertExpectations(t)
	})

	t.Run("Failure - Node Not Found", func(t *testing.T) {
		updateReq := &model.UpdateWorkflowNodeDTO{
			WorkflowNodeID: nodeID,
			State:          model.WorkflowNodeStateInProgress,
		}

		sqlMock.ExpectBegin()

		mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(nil, errors.New("not found")).Once()

		sqlMock.ExpectRollback()

		_, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve workflow node")
	})

	t.Run("Success - Transition to Completed (All Nodes Done)", func(t *testing.T) {
		updateReq := &model.UpdateWorkflowNodeDTO{
			WorkflowNodeID:      nodeID,
			State:               model.WorkflowNodeStateCompleted,
			AppendGlobalContext: map[string]any{"newKey": "newValue"},
		}

		// Initial Node State
		node := &model.WorkflowNode{
			BaseModel:        model.BaseModel{ID: nodeID},
			PreConsignmentID: &pcID,
			State:            model.WorkflowNodeStateInProgress,
		}

		sqlMock.ExpectBegin()

		// Get Node In Tx
		mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(node, nil).Once()

		// Append Context (Pre-transition)
		// First(pre_consignment)
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_context", "trader_id"}).AddRow(pcID, []byte(`{"initial": "val"}`), "trader1"))

		// Save(pre_consignment) - Update TraderContext
		sqlMock.ExpectExec(`UPDATE "pre_consignments"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Transition To Completed (StateMachine)
		// We need to mock the dependencies that StateMachine calls:
		// - UpdateWorkflowNodesInTx (Node -> Completed)
		// - CountIncompleteNodes... (Check if all done)
		// - UnlockDependentNodes... (Update siblings/dependents)

		// Update Node State
		mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 1 && nodes[0].State == model.WorkflowNodeStateCompleted
		})).Return(nil).Once()

		// Unlock Dependent Nodes (None for now)
		// It calls GetWorkflowNodesByPreConsignmentIDInTx to find siblings
		mockNodeRepo.On("GetWorkflowNodesByPreConsignmentIDInTx", ctx, mock.Anything, pcID).Return([]model.WorkflowNode{}, nil).Once()

		// Mark PreConsignment Completed (Service Logic)
		// First(pre_consignment)
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "state", "trader_id", "trader_context"}).
				AddRow(pcID, "IN_PROGRESS", "trader1", []byte(`{"initial": "val", "newKey": "newValue"}`)))

		// Save(pre_consignment) -> State = COMPLETED
		sqlMock.ExpectExec(`UPDATE "pre_consignments"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Sync Context to Auth (Service Logic)
		// First(auth.TraderContext) - Assume not found or found. Let's say found.
		// GORM adds FOR UPDATE due to Locking clause
		sqlMock.ExpectQuery(`SELECT \* FROM "trader_contexts" WHERE trader_id = \$1 ORDER BY "trader_contexts"."trader_id" LIMIT \$2 FOR UPDATE`).
			WithArgs("trader1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "trader_context"}).
				AddRow(uuid.New(), "trader1", []byte(`{}`)))

		// Save(auth.TraderContext)
		sqlMock.ExpectExec(`UPDATE "trader_contexts"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		sqlMock.ExpectCommit()

		_, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
		assert.NoError(t, err)

		mockNodeRepo.AssertExpectations(t)
	})
}

func TestPreConsignmentService_GetPreConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"

	t.Run("Success", func(t *testing.T) {
		pcID := uuid.New()
		templateID := uuid.New()

		// Expectation: Find
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
			WithArgs(traderID, model.PreConsignmentStateLocked).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
				AddRow(pcID, traderID, "IN_PROGRESS", templateID))

		// Preload 1: PreConsignmentTemplate
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
			WithArgs(templateID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Test PC Template"))

		// Preload 2: WorkflowNodes
		sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."pre_consignment_id" = \$1`).
			WithArgs(pcID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id"}))

		results, err := service.GetPreConsignmentsByTraderID(ctx, traderID)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, pcID, results[0].ID)
	})

	t.Run("Empty", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
			WithArgs(traderID, model.PreConsignmentStateLocked).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}))

		results, err := service.GetPreConsignmentsByTraderID(ctx, traderID)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestPreConsignmentService_SyncTraderContext(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	pcID := uuid.New()
	traderID := "trader1"

	// We can't access syncTraderContextToAuth directly as it's private,
	// but we can trigger it via markPreConsignmentAsCompleted, which is also private,
	// BUT markPreConsignmentAsCompleted is called when a node transition completes the pre-consignment.

	// So we act out a node transition that completes the pre-consignment.

	nodeID := uuid.New()
	updateReq := &model.UpdateWorkflowNodeDTO{
		WorkflowNodeID:      nodeID,
		State:               model.WorkflowNodeStateCompleted,
		AppendGlobalContext: map[string]any{"newKey": "newValue"},
	}

	node := &model.WorkflowNode{
		BaseModel:        model.BaseModel{ID: nodeID},
		PreConsignmentID: &pcID,
		State:            model.WorkflowNodeStateInProgress,
	}

	sqlMock.ExpectBegin()

	// Get Node
	mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(node, nil).Once()

	// Append Context
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(pcID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_context", "trader_id"}).
			AddRow(pcID, []byte(`{"initial": "val"}`), traderID))

	sqlMock.ExpectExec(`UPDATE "pre_consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	// Transition (Completed)
	mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return(nil).Once()

	// Sibling nodes (return just this one, so all completed)
	mockNodeRepo.On("GetWorkflowNodesByPreConsignmentIDInTx", ctx, mock.Anything, pcID).Return([]model.WorkflowNode{*node}, nil).Once()

	// Mark PreConsignment Completed
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(pcID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "trader_id", "trader_context"}).
			AddRow(pcID, "IN_PROGRESS", traderID, []byte(`{"initial": "val", "newKey": "newValue"}`)))

	sqlMock.ExpectExec(`UPDATE "pre_consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	// Sync Trader Context (The part we really want to test)
	// Query TraderContext FOR UPDATE
	sqlMock.ExpectQuery(`SELECT \* FROM "trader_contexts" WHERE trader_id = \$1 ORDER BY "trader_contexts"."trader_id" LIMIT \$2 FOR UPDATE`).
		WithArgs(traderID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "trader_context"}).
			AddRow(uuid.New(), traderID, []byte(`{"existing": "data"}`)))

	// Update TraderContext
	// Should merge {"initial": "val", "newKey": "newValue"} into {"existing": "data"}
	sqlMock.ExpectExec(`UPDATE "trader_contexts"`).WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	_, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
	assert.NoError(t, err)
}

func TestPreConsignmentService_InitializePreConsignment_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	templateID := uuid.New()
	createReq := &model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}

	t.Run("Template Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1 ORDER BY "pre_consignment_templates"."id" LIMIT \$2`).
			WithArgs(templateID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		resp, nodes, err := service.InitializePreConsignment(ctx, createReq, "trader1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Nil(t, resp)
		assert.Nil(t, nodes)
	})

	t.Run("Workflow Template Not Found", func(t *testing.T) {
		workflowTemplateID := uuid.New()
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1 ORDER BY "pre_consignment_templates"."id" LIMIT \$2`).
			WithArgs(templateID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
				AddRow(templateID, workflowTemplateID, []byte("[]")))

		mockTemplateProvider.On("GetWorkflowTemplateByID", ctx, workflowTemplateID).Return(nil, errors.New("wf error")).Once()

		resp, nodes, err := service.InitializePreConsignment(ctx, createReq, "trader1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get workflow template")
		assert.Nil(t, resp)
		assert.Nil(t, nodes)
	})
}

func TestPreConsignmentService_GetTraderPreConsignments_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewPreConsignmentService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	t.Run("Count Error", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT count\\(\*\\) FROM "pre_consignment_templates"`).
			WillReturnError(errors.New("db error"))

		result, err := service.GetTraderPreConsignments(ctx, traderID, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, model.TraderPreConsignmentsResponseDTO{}, result)
	})
}

func TestPreConsignmentService_UpdateWorkflowNodeState_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewPreConsignmentService(db, nil, mockNodeRepo)
	ctx := context.Background()
	nodeID := uuid.New()
	updateReq := &model.UpdateWorkflowNodeDTO{
		WorkflowNodeID: nodeID,
		State:          model.WorkflowNodeStateInProgress,
	}

	t.Run("DB Error on Node Retrieval", func(t *testing.T) {
		sqlMock.ExpectBegin()
		mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(nil, errors.New("db error")).Once()
		sqlMock.ExpectRollback()

		nodes, context, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, nodes)
		assert.Nil(t, context)
	})
}
