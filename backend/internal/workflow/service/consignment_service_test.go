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

// MockTemplateProvider
type MockTemplateProvider struct {
	mock.Mock
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNodeTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
}

func TestConsignmentService_InitializeConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	hsCodeID := uuid.New()
	createReq := &model.CreateConsignmentDTO{
		Flow: model.ConsignmentFlowImport,
		Items: []model.CreateConsignmentItemDTO{
			{HSCodeID: hsCodeID},
		},
	}
	globalContext := map[string]any{"key": "value"}

	// Mock Template Provider
	workflowTemplate := &model.WorkflowTemplate{
		BaseModel:     model.BaseModel{ID: uuid.New()},
		Name:          "Test Template",
		NodeTemplates: model.UUIDArray{uuid.New()},
	}
	mockTemplateProvider.On("GetWorkflowTemplateByHSCodeIDAndFlow", ctx, hsCodeID, model.ConsignmentFlowImport).Return(workflowTemplate, nil)

	// Mock Template Provider for creating nodes
	nodeTemplate := model.WorkflowNodeTemplate{
		BaseModel: model.BaseModel{ID: workflowTemplate.NodeTemplates[0]},
		Name:      "Test Node Template",
		Type:      "SIMPLE_FORM",
	}
	mockTemplateProvider.On("GetWorkflowNodeTemplatesByIDs", ctx, []uuid.UUID{nodeTemplate.ID}).Return([]model.WorkflowNodeTemplate{nodeTemplate}, nil)

	// Mock Node Repo for creating nodes
	createdNodes := []model.WorkflowNode{
		{
			BaseModel:              model.BaseModel{ID: uuid.New()},
			WorkflowNodeTemplateID: nodeTemplate.ID,
			State:                  model.WorkflowNodeStateLocked, // Initial state before resolving dependencies
		},
	}
	// Note: We need to match the arguments loosely or precisely.
	// Here we just test the flow, so we expect CreateWorkflowNodesInTx call.
	mockNodeRepo.On("CreateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return(createdNodes, nil)
	// UpdateWorkflowNodesInTx will be called to update node states (e.g. to READY)
	mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return(nil)

	// Mock DB Expectations
	sqlMock.ExpectBegin()
	// Create Consignment
	// GORM might use Exec if it doesn't need to return generated values (since we calculate UUID in BeforeCreate)
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Create Workflow Nodes
	// Since we mock the NodeRepo, we don't expect DB calls for nodes here,
	// BUT the service calls createWorkflowNodesInTx with 'tx'.
	// The mockRepo uses the passed 'tx'. If the mockRepo implementation in the test
	// just returns, it doesn't touch the DB.
	// However, we passed the *real* Gorm DB (which is mocked underneath) to the service.
	// The service starts a valid transaction on it.

	sqlMock.ExpectCommit()

	// Select Consignment
	// Gorm adds "id = <id>" from struct and "id = <id>" from condition, plus LIMIT 1
	consignmentID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	// Select WorkflowNodes (Preload)
	// Expectation for Preload WorkflowNodes
	// It usually selects nodes where consignment_id IN (...)
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."consignment_id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id", "state", "consignment_id"}).
			AddRow(uuid.New(), nodeTemplate.ID, "READY", consignmentID))

		// Select WorkflowNodeTemplates (Nested Preload)
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE "workflow_node_templates"."id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type"}).
			AddRow(nodeTemplate.ID, "Test Node Template", "SIMPLE_FORM"))

	// Batch Load HS Codes
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	// Run Test
	resp, nodes, err := service.InitializeConsignment(ctx, createReq, traderID, globalContext)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, nodes) // Should return the ready nodes (which might be the createdNodes updated to READY)

	mockTemplateProvider.AssertExpectations(t)
	mockNodeRepo.AssertExpectations(t)
}

func TestConsignmentService_UpdateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	consignmentID := uuid.New()

	state := model.ConsignmentStateFinished
	updateReq := &model.UpdateConsignmentDTO{
		ConsignmentID: consignmentID,
		State:         &state,
	}

	// First: Retrieve consignment
	// Gorm adds LIMIT 1
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	// Updates
	// Gorm wraps updates in transaction by default
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments" SET "state"=\$1,"updated_at"=\$2 WHERE "id" = \$3`).
		WithArgs("FINISHED", sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Reload (Preload)
	// Consignment
	hsCodeID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "FINISHED", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	// WorkflowNodes
	nodeTemplateID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."consignment_id" = \$1`).
		WithArgs(consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id", "state", "consignment_id"}).
			AddRow(uuid.New(), nodeTemplateID, "COMPLETED", consignmentID))

		// Templates
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE "workflow_node_templates"."id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type"}).
			AddRow(nodeTemplateID, "Test Node Template", "SIMPLE_FORM"))

	// Expectation for Batch Load HS Codes
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	resp, err := service.UpdateConsignment(ctx, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, model.ConsignmentStateFinished, resp.State)
}

func TestConsignmentService_UpdateWorkflowNodeStateAndPropagateChanges(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	nodeID := uuid.New()
	consignmentID := uuid.New()

	updateReq := &model.UpdateWorkflowNodeDTO{
		WorkflowNodeID: nodeID,
		State:          model.WorkflowNodeStateInProgress,
	}

	node := &model.WorkflowNode{
		BaseModel:     model.BaseModel{ID: nodeID},
		ConsignmentID: &consignmentID,
		State:         model.WorkflowNodeStateReady,
	}

	sqlMock.ExpectBegin()

	// Get Workflow Node (In Tx)
	mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(node, nil)

	// Transition (In Progress) -> Updates Node
	// State machine calls UpdateWorkflowNodesInTx
	mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
		return len(nodes) == 1 && nodes[0].State == model.WorkflowNodeStateInProgress
	})).Return(nil)

	// Append Global Context
	// First(consignment)
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "global_context"}).AddRow(consignmentID, []byte("{}")))

	// Save(consignment)
	// Save updates all fields
	sqlMock.ExpectExec(`UPDATE "consignments" SET "created_at"=\$1,"updated_at"=\$2,"flow"=\$3,"trader_id"=\$4,"state"=\$5,"items"=\$6,"global_context"=\$7,"end_node_id"=\$8 WHERE "id" = \$9`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	newReadyNodes, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
	assert.NoError(t, err)
	assert.Empty(t, newReadyNodes) // Transition to InProgress doesn't unlock dependent nodes
}

func TestConsignmentService_GetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	// We don't need these mocks for this test but NewConsignmentService requires them
	// We can pass nil if we don't trigger methods using them, or pass mocks.
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	consignmentID := uuid.New()

	// Expectation for Find (Consignments with Preload)
	hsCodeID := uuid.New()
	// Select Consignments
	// Gorm First adds ORDER BY and LIMIT
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

		// Select WorkflowNodes (Preload)
	nodeTemplateID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."consignment_id" = \$1`).
		WithArgs(consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id", "state", "consignment_id"}).
			AddRow(uuid.New(), nodeTemplateID, "READY", consignmentID))

	// Select WorkflowNodeTemplates (Nested Preload)
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE "workflow_node_templates"."id" = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type"}).
			AddRow(nodeTemplateID, "Test Node Template", "SIMPLE_FORM"))

	// Expectation for Batch Load HS Codes
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	// Run Test
	result, err := service.GetConsignmentByID(ctx, consignmentID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	assert.Len(t, result.WorkflowNodes, 1)
}

func TestConsignmentService_GetConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	// We don't need these mocks for this test but NewConsignmentService requires them
	// We can pass nil if we don't trigger methods using them, or pass mocks.
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	limit := 10
	offset := 0
	filter := model.ConsignmentFilter{}

	// Expectation for Count
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Expectation for Find (Consignments with Preload)
	consignmentID := uuid.New()
	hsCodeID := uuid.New()
	// Select Consignments
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1 ORDER BY created_at DESC LIMIT \$2`).
		WithArgs(traderID, limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	// Select WorkflowNodes (Preload)
	sqlMock.ExpectQuery(`SELECT consignment_id, count\(\*\) as total, count\(case when state = \$1 then 1 end\) as completed FROM "workflow_nodes" WHERE consignment_id IN \(\$2\) GROUP BY "consignment_id"`).
		WithArgs(sqlmock.AnyArg(), consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"consignment_id", "total", "completed"}).AddRow(consignmentID, 1, 0))

	// Expectation for Batch Load HS Codes
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	// Run Test
	result, err := service.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, consignmentID, result.Items[0].ID)
	// Check WorkflowNodes is not asserted as it's not present in SummaryDTO

}

func TestConsignmentService_UpdateWorkflowNodeState_Completion(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	nodeID := uuid.New()
	consignmentID := uuid.New()

	updateReq := &model.UpdateWorkflowNodeDTO{
		WorkflowNodeID: nodeID,
		State:          model.WorkflowNodeStateCompleted,
	}

	node := &model.WorkflowNode{
		BaseModel:     model.BaseModel{ID: nodeID},
		ConsignmentID: &consignmentID,
		State:         model.WorkflowNodeStateInProgress,
	}

	sqlMock.ExpectBegin()

	// Get Workflow Node (In Tx)
	mockNodeRepo.On("GetWorkflowNodeByIDInTx", ctx, mock.Anything, nodeID).Return(node, nil).Once()

	// The new SELECT consignment for EndNodeID
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	// Transition (Completed)
	// Update Node State
	mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
		return len(nodes) == 1 && nodes[0].State == model.WorkflowNodeStateCompleted
	})).Return(nil).Once()

	// Get Siblings (Check all nodes completed)
	siblingNodes := []model.WorkflowNode{*node}
	mockNodeRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, mock.Anything, consignmentID).Return(siblingNodes, nil).Once()

	// Mark Consignment As Finished
	// First(consignment) - Returns IN_PROGRESS
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	// Save(consignment) -> State = FINISHED
	sqlMock.ExpectExec(`UPDATE "consignments" SET "created_at"=\$1,"updated_at"=\$2,"flow"=\$3,"trader_id"=\$4,"state"=\$5,"items"=\$6,"global_context"=\$7,"end_node_id"=\$8 WHERE "id" = \$9`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "FINISHED", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Append Global Context
	// First(consignment) - Returns FINISHED (simulating DB state after previous update)
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "global_context"}).AddRow(consignmentID, "FINISHED", []byte("{}")))

	// Save(consignment) - Updates Global Context, State should remain FINISHED
	sqlMock.ExpectExec(`UPDATE "consignments" SET "created_at"=\$1,"updated_at"=\$2,"flow"=\$3,"trader_id"=\$4,"state"=\$5,"items"=\$6,"global_context"=\$7,"end_node_id"=\$8 WHERE "id" = \$9`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "FINISHED", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	newReadyNodes, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
	assert.NoError(t, err)
	assert.Empty(t, newReadyNodes) // No dependents
}

func TestConsignmentService_InitializeConsignment_Failure(t *testing.T) {
	db, _ := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	hsCodeID := uuid.New()
	createReq := &model.CreateConsignmentDTO{
		Flow: model.ConsignmentFlowImport,
		Items: []model.CreateConsignmentItemDTO{
			{HSCodeID: hsCodeID},
		},
	}

	t.Run("Template Not Found", func(t *testing.T) {
		mockTemplateProvider.On("GetWorkflowTemplateByHSCodeIDAndFlow", ctx, hsCodeID, model.ConsignmentFlowImport).Return(nil, errors.New("template not found")).Once()

		resp, nodes, err := service.InitializeConsignment(ctx, createReq, "trader1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get workflow template")
		assert.Nil(t, resp)
		assert.Nil(t, nodes)
	})

	t.Run("Node Templates Fetch Error", func(t *testing.T) {
		db, sqlMock := setupTestDB(t)
		mockTemplateProvider := new(MockTemplateProvider)
		mockNodeRepo := new(MockWorkflowNodeRepository)
		service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

		localHSCodeID := uuid.New()
		localCreateReq := &model.CreateConsignmentDTO{
			Flow: model.ConsignmentFlowImport,
			Items: []model.CreateConsignmentItemDTO{
				{HSCodeID: localHSCodeID},
			},
		}

		workflowTemplate := &model.WorkflowTemplate{
			BaseModel:     model.BaseModel{ID: uuid.New()},
			NodeTemplates: model.UUIDArray{uuid.New()},
		}
		mockTemplateProvider.On("GetWorkflowTemplateByHSCodeIDAndFlow", mock.Anything, localHSCodeID, model.ConsignmentFlowImport).Return(workflowTemplate, nil).Once()

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`INSERT INTO "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

		mockTemplateProvider.On("GetWorkflowNodeTemplatesByIDs", mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
			return len(ids) == 1 && ids[0] == workflowTemplate.NodeTemplates[0]
		})).Return(nil, errors.New("fetch error")).Once()
		sqlMock.ExpectRollback()

		resp, nodes, err := service.InitializeConsignment(context.Background(), localCreateReq, "trader1", nil)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "failed to create workflow nodes")
			assert.Contains(t, err.Error(), "failed to retrieve workflow node templates")
		}
		assert.Nil(t, resp)
		assert.Nil(t, nodes)
		sqlMock.ExpectationsWereMet()
	})
}

func TestConsignmentService_UpdateConsignment_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	consignmentID := uuid.New()
	state := model.ConsignmentStateFinished
	updateReq := &model.UpdateConsignmentDTO{
		ConsignmentID: consignmentID,
		State:         &state,
	}

	t.Run("Consignment Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
			WithArgs(consignmentID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		resp, err := service.UpdateConsignment(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Update DB Error", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
			WithArgs(consignmentID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnError(errors.New("db error"))
		sqlMock.ExpectRollback()

		resp, err := service.UpdateConsignment(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestConsignmentService_GetConsignmentsByTraderID_EdgeCases(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewConsignmentService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	t.Run("Empty Results", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
			WithArgs(traderID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1`).
			WithArgs(traderID, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		limit := 10
		offset := 0
		result, err := service.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, model.ConsignmentFilter{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.TotalCount)
		assert.Empty(t, result.Items)
	})

	t.Run("Count Error", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
			WillReturnError(errors.New("count error"))

		limit := 10
		offset := 0
		result, err := service.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, model.ConsignmentFilter{})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
