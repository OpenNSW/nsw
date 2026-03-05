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

	"github.com/OpenNSW/nsw/internal/auth"
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

func (m *MockTemplateProvider) GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
}

func TestConsignmentService_CreateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	chaID := uuid.New()
	createReq := &model.CreateConsignmentDTO{
		Flow:  model.ConsignmentFlowImport,
		CHAID: &chaID,
	}
	globalContext := map[string]any{"key": "value"}

	// Mock DB Expectations for CreateConsignment
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Select Consignment (Reload)
	consignmentID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 AND "consignments"\."id" = \$2 ORDER BY "consignments"."id" LIMIT \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "cha_id"}).
			AddRow(consignmentID, "IMPORT", "trader1", "AWAITING_INITIATION", chaID))

	sqlMock.ExpectQuery(`SELECT \* FROM "customs_house_agents" WHERE "customs_house_agents"."id" = \$1`).
		WithArgs(chaID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(chaID, "Test Agency"))

	// Run Test
	resp, err := service.CreateConsignment(ctx, createReq, traderID, globalContext)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, model.ConsignmentStateAwaitingInitiation, resp.State)

	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_InitializeConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	consignmentID := uuid.New()
	hsCodeID := uuid.New()
	initReq := &model.InitializeConsignmentDTO{
		Items: []model.CreateConsignmentItemDTO{
			{HSCodeID: hsCodeID},
		},
	}

	// Mock DB Expectations for InitializeConsignment
	sqlMock.ExpectBegin()
	// Lock for update
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"\."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "state"}).
			AddRow(consignmentID, model.ConsignmentFlowImport, model.ConsignmentStateAwaitingInitiation))

	// Mock Template Provided
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
			State:                  model.WorkflowNodeStateLocked,
		},
	}
	mockNodeRepo.On("CreateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return(createdNodes, nil)
	mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return(nil)

	// Update Consignment
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	// Reload (Preload)
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 AND "consignments"\."id" = \$2 ORDER BY "consignments"."id" LIMIT \$3`).
		WithArgs(consignmentID, consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	// Preloads
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."consignment_id" = \$1`).
		WithArgs(consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id", "consignment_id"}).AddRow(uuid.New(), nodeTemplate.ID, consignmentID))
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE "workflow_node_templates"."id" = \$1`).
		WithArgs(nodeTemplate.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(nodeTemplate.ID, "Test"))

	// HS Code Load
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

	// Run Test
	resp, nodes, err := service.InitializeConsignment(ctx, consignmentID, initReq)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, model.ConsignmentStateInProgress, resp.State)
	assert.NotEmpty(t, nodes)

	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_UpdateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	consignmentID := uuid.New()
	traderID := "trader1"
	ctx = context.WithValue(ctx, auth.AuthContextKey, &auth.AuthContext{TraderContext: &auth.TraderContext{TraderID: traderID}})

	state := model.ConsignmentStateFinished
	updateReq := &model.UpdateConsignmentDTO{
		ConsignmentID: consignmentID,
		State:         &state,
	}

	// First: Retrieve consignment
	// Gorm adds LIMIT 1
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1 AND id = \$2 ORDER BY "consignments"."id" LIMIT \$3`).
		WithArgs(traderID, consignmentID, 1).
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
	chaID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 AND "consignments"\."id" = \$2 ORDER BY "consignments"\."id" LIMIT \$3`).
		WithArgs(consignmentID, consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "cha_id", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, "FINISHED", time.Now(), time.Now(), chaID, []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	sqlMock.ExpectQuery(`SELECT \* FROM "customs_house_agents" WHERE "customs_house_agents"\."id" = \$1`).
		WithArgs(chaID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).AddRow(chaID, "Test CHA", "A CHA agent"))

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
	assert.NoError(t, sqlMock.ExpectationsWereMet())
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
	sqlMock.ExpectExec(`UPDATE "consignments" SET "created_at"=\$1,"updated_at"=\$2,"flow"=\$3,"trader_id"=\$4,"state"=\$5,"items"=\$6,"global_context"=\$7,"end_node_id"=\$8,"cha_id"=\$9 WHERE "id" = \$10`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	newReadyNodes, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
	assert.NoError(t, err)
	assert.Empty(t, newReadyNodes) // Transition to InProgress doesn't unlock dependent nodes
	assert.NoError(t, sqlMock.ExpectationsWereMet())
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
	traderID := "trader1"
	ctx = context.WithValue(ctx, auth.AuthContextKey, &auth.AuthContext{TraderContext: &auth.TraderContext{TraderID: traderID}})

	// Expectation for Find (Consignments with Preload)
	hsCodeID := uuid.New()
	chaID := uuid.New()
	// Select Consignments
	// Gorm First adds ORDER BY and LIMIT
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1 AND id = \$2 ORDER BY "consignments"\."id" LIMIT \$3`).
		WithArgs(traderID, consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "cha_id", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, "IN_PROGRESS", time.Now(), time.Now(), chaID, []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	sqlMock.ExpectQuery(`SELECT \* FROM "customs_house_agents" WHERE "customs_house_agents"\."id" = \$1`).
		WithArgs(chaID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).AddRow(chaID, "Test CHA", "A CHA agent"))

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
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignments(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	// We don't need these mocks for this test but NewConsignmentService requires them
	// We can pass nil if we don't trigger methods using them, or pass mocks.
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	ctx = context.WithValue(ctx, auth.AuthContextKey, &auth.AuthContext{TraderContext: &auth.TraderContext{TraderID: traderID}})

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
	result, err := service.GetConsignments(ctx, &offset, &limit, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, consignmentID, result.Items[0].ID)
	// Check WorkflowNodes is not asserted as it's not present in SummaryDTO
	assert.NoError(t, sqlMock.ExpectationsWereMet())

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
	completedSibling := *node
	completedSibling.State = model.WorkflowNodeStateCompleted
	siblingNodes := []model.WorkflowNode{completedSibling}
	mockNodeRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, mock.Anything, consignmentID).Return(siblingNodes, nil).Once()

	// Mark Consignment As Finished
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	// Save(consignment) -> State = FINISHED
	sqlMock.ExpectExec(`UPDATE "consignments" SET "created_at"=\$1,"updated_at"=\$2,"flow"=\$3,"trader_id"=\$4,"state"=\$5,"items"=\$6,"global_context"=\$7,"end_node_id"=\$8,"cha_id"=\$9 WHERE "id" = \$10`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "FINISHED", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Append Global Context
	// First(consignment)
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "global_context"}).AddRow(consignmentID, "FINISHED", []byte("{}")))

	// Save(consignment) - Updates Global Context, State should remain FINISHED
	sqlMock.ExpectExec(`UPDATE "consignments" SET "created_at"=\$1,"updated_at"=\$2,"flow"=\$3,"trader_id"=\$4,"state"=\$5,"items"=\$6,"global_context"=\$7,"end_node_id"=\$8,"cha_id"=\$9 WHERE "id" = \$10`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "FINISHED", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), consignmentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	newReadyNodes, _, err := service.UpdateWorkflowNodeStateAndPropagateChanges(ctx, updateReq)
	assert.NoError(t, err)
	assert.Empty(t, newReadyNodes) // No dependents
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_InitializeConsignment_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)

	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)
	ctx := context.Background()
	hsCodeID := uuid.New()

	t.Run("Template Not Found", func(t *testing.T) {
		consignmentID := uuid.New()
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"\."id" LIMIT \$2`).
			WithArgs(consignmentID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "state"}).
				AddRow(consignmentID, model.ConsignmentFlowImport, model.ConsignmentStateAwaitingInitiation))

		mockTemplateProvider.On("GetWorkflowTemplateByHSCodeIDAndFlow", ctx, hsCodeID, model.ConsignmentFlowImport).Return(nil, errors.New("template not found")).Once()
		sqlMock.ExpectRollback()

		resp, nodes, err := service.InitializeConsignment(ctx, consignmentID, &model.InitializeConsignmentDTO{Items: []model.CreateConsignmentItemDTO{{HSCodeID: hsCodeID}}})
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

		consignmentID := uuid.New()

		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"\."id" LIMIT \$2`).
			WithArgs(consignmentID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "state"}).
				AddRow(consignmentID, model.ConsignmentFlowImport, model.ConsignmentStateAwaitingInitiation))

		localHSCodeID := uuid.New()
		workflowTemplate := &model.WorkflowTemplate{
			BaseModel:     model.BaseModel{ID: uuid.New()},
			NodeTemplates: model.UUIDArray{uuid.New()},
		}
		mockTemplateProvider.On("GetWorkflowTemplateByHSCodeIDAndFlow", mock.Anything, localHSCodeID, model.ConsignmentFlowImport).Return(workflowTemplate, nil).Once()

		mockTemplateProvider.On("GetWorkflowNodeTemplatesByIDs", mock.Anything, mock.MatchedBy(func(ids []uuid.UUID) bool {
			return len(ids) == 1 && ids[0] == workflowTemplate.NodeTemplates[0]
		})).Return(nil, errors.New("fetch error")).Once()
		sqlMock.ExpectRollback()

		resp, nodes, err := service.InitializeConsignment(context.Background(), consignmentID, &model.InitializeConsignmentDTO{Items: []model.CreateConsignmentItemDTO{{HSCodeID: localHSCodeID}}})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "failed to create workflow nodes")
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
	traderID := "trader1"
	ctx = context.WithValue(ctx, auth.AuthContextKey, &auth.AuthContext{TraderContext: &auth.TraderContext{TraderID: traderID}})
	state := model.ConsignmentStateFinished
	updateReq := &model.UpdateConsignmentDTO{
		ConsignmentID: consignmentID,
		State:         &state,
	}

	t.Run("Consignment Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1 AND id = \$2 ORDER BY "consignments"."id" LIMIT \$3`).
			WithArgs(traderID, consignmentID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		resp, err := service.UpdateConsignment(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Update DB Error", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE trader_id = \$1 AND id = \$2 ORDER BY "consignments"."id" LIMIT \$3`).
			WithArgs(traderID, consignmentID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnError(errors.New("db error"))
		sqlMock.ExpectRollback()

		resp, err := service.UpdateConsignment(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestConsignmentService_GetConsignments_EdgeCases(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewConsignmentService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"
	ctx = context.WithValue(ctx, auth.AuthContextKey, &auth.AuthContext{TraderContext: &auth.TraderContext{TraderID: traderID}})

	t.Run("Empty Results", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
			WithArgs(traderID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		limit := 10
		offset := 0
		result, err := service.GetConsignments(ctx, &offset, &limit, model.ConsignmentFilter{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.TotalCount)
		assert.Empty(t, result.Items)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Count Error", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
			WillReturnError(errors.New("count error"))

		limit := 10
		offset := 0
		result, err := service.GetConsignments(ctx, &offset, &limit, model.ConsignmentFilter{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
func TestConsignmentService_InitializeConsignment_NoCHA(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	consignmentID := uuid.New()
	hsCodeID := uuid.New()
	nodeTemplateID := uuid.New()

	mockTemplateProvider.On("GetWorkflowTemplateByHSCodeIDAndFlow", ctx, hsCodeID, model.ConsignmentFlowImport).Return(&model.WorkflowTemplate{
		BaseModel:     model.BaseModel{ID: uuid.New()},
		NodeTemplates: model.UUIDArray{nodeTemplateID},
	}, nil)
	mockTemplateProvider.On("GetWorkflowNodeTemplatesByIDs", ctx, []uuid.UUID{nodeTemplateID}).Return([]model.WorkflowNodeTemplate{{BaseModel: model.BaseModel{ID: nodeTemplateID}}}, nil)
	mockNodeRepo.On("CreateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return([]model.WorkflowNode{{BaseModel: model.BaseModel{ID: uuid.New()}, WorkflowNodeTemplateID: nodeTemplateID}}, nil)
	mockNodeRepo.On("UpdateWorkflowNodesInTx", ctx, mock.Anything, mock.Anything).Return(nil)

	// Existing consignment in AWAITING_INITIATION with no CHA (cha_id nil)
	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"\."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "cha_id"}).
			AddRow(consignmentID, model.ConsignmentFlowImport, traderID, model.ConsignmentStateAwaitingInitiation, nil))

	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 AND "consignments"\."id" = \$2 ORDER BY "consignments"."id" LIMIT \$3`).
		WithArgs(consignmentID, consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "cha_id", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, "IN_PROGRESS", nil, []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE "workflow_nodes"."consignment_id" = \$1`).
		WithArgs(consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id", "consignment_id"}).AddRow(uuid.New(), nodeTemplateID, consignmentID))
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE "workflow_node_templates"."id" = \$1`).
		WithArgs(nodeTemplateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(nodeTemplateID, "Test"))
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

	initReq := &model.InitializeConsignmentDTO{Items: []model.CreateConsignmentItemDTO{{HSCodeID: hsCodeID}}}
	resp, _, err := service.InitializeConsignment(ctx, consignmentID, initReq)
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Nil(t, resp.CHAID)
	}
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_CreateConsignment_FKViolation(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTemplateProvider := new(MockTemplateProvider)
	mockNodeRepo := new(MockWorkflowNodeRepository)
	service := NewConsignmentService(db, mockTemplateProvider, mockNodeRepo)

	ctx := context.Background()
	traderID := "trader1"
	nonExistentCHAID := uuid.New()
	createReq := &model.CreateConsignmentDTO{
		Flow:  model.ConsignmentFlowImport,
		CHAID: &nonExistentCHAID,
	}

	// CreateConsignment may run in a transaction; FK violation on cha_id
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).
		WillReturnError(errors.New("insert or update on table \"consignments\" violates foreign key constraint \"consignments_cha_id_fkey\""))
	sqlMock.ExpectRollback()

	resp, err := service.CreateConsignment(ctx, createReq, traderID, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "violates foreign key constraint")
	assert.Nil(t, resp)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignments_CHA(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewConsignmentService(db, nil, nil)

	ctx := context.Background()
	chaID := uuid.New().String()
	// Mock CHA Auth Context
	ctx = context.WithValue(ctx, auth.AuthContextKey, &auth.AuthContext{
		Role:     "CHA",
		AgencyID: chaID,
		// TraderContext can be nil since we now have nil check in GetTraderID,
		// but providing it for completeness if needed.
	})

	limit := 10
	offset := 0

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE cha_id = \$1`).
		WithArgs(chaID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	consignmentID := uuid.New()
	hsCodeID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE cha_id = \$1 ORDER BY created_at DESC LIMIT \$2`).
		WithArgs(chaID, limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "cha_id", "items"}).
			AddRow(consignmentID, "trader1", chaID, []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	sqlMock.ExpectQuery(`SELECT consignment_id, count\(\*\) as total, count\(case when state = \$1 then 1 end\) as completed FROM "workflow_nodes" WHERE consignment_id IN \(\$2\) GROUP BY "consignment_id"`).
		WithArgs(model.WorkflowNodeStateCompleted, consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"consignment_id", "total", "completed"}).AddRow(consignmentID, 5, 2))

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN \(\$1\)`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

	result, err := service.GetConsignments(ctx, &offset, &limit, model.ConsignmentFilter{})
	assert.NoError(t, err)
	if assert.NotNil(t, result) {
		assert.Equal(t, int64(1), result.TotalCount)
		assert.Equal(t, consignmentID, result.Items[0].ID)
	}
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
