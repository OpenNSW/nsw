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

	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	workflowManagerV1 "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// MockTemplateProvider implements TemplateProvider for testing.
type MockTemplateProvider struct {
	mock.Mock
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlowV2(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplateV2, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplateV2), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByID(ctx context.Context, id string) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []string) ([]model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNodeTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error) {
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

// MockWorkflowManager implements manager.Manager for testing.
type MockWorkflowManager struct {
	mock.Mock
}

func (m *MockWorkflowManager) StartWorkflowInstance(ctx context.Context, tx *gorm.DB, workflowID string, workflowTemplates []model.WorkflowTemplate, globalContext map[string]any, handler workflowManagerV1.WorkflowEventHandler) error {
	args := m.Called(ctx, tx, workflowID, workflowTemplates, globalContext, handler)
	return args.Error(0)
}

func (m *MockWorkflowManager) GetWorkflowInstance(ctx context.Context, workflowID string) (*model.Workflow, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Workflow), args.Error(1)
}

func (m *MockWorkflowManager) RegisterTaskHandler(_ workflowManagerV1.TaskInitHandler) error {
	return nil
}

func (m *MockWorkflowManager) HandleTaskUpdate(_ context.Context, _ taskManager.WorkflowManagerNotification) error {
	return nil
}

// MockWMV2 implements workflowManagerV2.TemporalManager for testing.
type MockWMV2 struct {
	mock.Mock
}

func (m *MockWMV2) StartWorkflow(ctx context.Context, ID string, jsonDSL []byte, initialWorkflowVariables map[string]any) error {
	args := m.Called(ctx, ID, jsonDSL, initialWorkflowVariables)
	return args.Error(0)
}

func (m *MockWMV2) TaskDone(ctx context.Context, workflowID, runID, nodeID string, output map[string]any) error {
	args := m.Called(ctx, workflowID, runID, nodeID, output)
	return args.Error(0)
}

func (m *MockWMV2) TaskUpdate(ctx context.Context, workflowID, runID string, update workflowManagerV2.UpdateEvent) error {
	args := m.Called(ctx, workflowID, runID, update)
	return args.Error(0)
}

func (m *MockWMV2) GetStatus(ctx context.Context, workflowID string) (*workflowManagerV2.WorkflowInstance, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowManagerV2.WorkflowInstance), args.Error(1)
}

func TestConsignmentService_GetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	// TODO: Add tests for workflow manager v2
	// mockWMV2 := new(MockWMV2)
	svc := NewConsignmentService(db, nil, mockWM, nil)

	ctx := context.Background()
	consignmentID := uuid.NewString()
	hsCodeID := uuid.NewString()
	nodeTemplateID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID+`"}]`)))

	mockWM.On("GetWorkflowInstance", ctx, consignmentID).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: consignmentID},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.NewString()},
				WorkflowNodeTemplateID: nodeTemplateID,
				State:                  model.WorkflowNodeStateReady,
				WorkflowNodeTemplate: model.WorkflowNodeTemplate{
					BaseModel: model.BaseModel{ID: nodeTemplateID},
					Name:      "Test Node Template",
					Type:      "SIMPLE_FORM",
				},
			},
		},
	}, nil)

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	result, err := svc.GetConsignmentByID(ctx, consignmentID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	assert.Len(t, result.WorkflowNodes, 1)
	mockWM.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_UpdateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	// TODO: Add tests for workflow manager v2
	// mockWMV2 := new(MockWMV2)
	svc := NewConsignmentService(db, nil, mockWM, nil)

	ctx := context.Background()
	consignmentID := uuid.NewString()
	hsCodeID := uuid.NewString()
	nodeTemplateID := uuid.NewString()

	state := model.ConsignmentStateFinished
	updateReq := &model.UpdateConsignmentDTO{
		ConsignmentID: consignmentID,
		State:         &state,
	}

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "FINISHED", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID+`"}]`)))

	mockWM.On("GetWorkflowInstance", ctx, consignmentID).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: consignmentID},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.NewString()},
				WorkflowNodeTemplateID: nodeTemplateID,
				State:                  model.WorkflowNodeStateCompleted,
				WorkflowNodeTemplate: model.WorkflowNodeTemplate{
					BaseModel: model.BaseModel{ID: nodeTemplateID},
					Name:      "Test Node Template",
					Type:      "SIMPLE_FORM",
				},
			},
		},
	}, nil)

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	resp, err := svc.UpdateConsignment(ctx, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, model.ConsignmentStateFinished, resp.State)
	mockWM.AssertExpectations(t)
}

func TestConsignmentService_UpdateConsignment_NotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewConsignmentService(db, nil, nil, nil)
	ctx := context.Background()
	consignmentID := uuid.NewString()
	state := model.ConsignmentStateFinished
	updateReq := &model.UpdateConsignmentDTO{
		ConsignmentID: consignmentID,
		State:         &state,
	}

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	resp, err := svc.UpdateConsignment(ctx, updateReq)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentsByTraderID_Empty(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewConsignmentService(db, nil, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	limit := 10
	offset := 0
	result, err := svc.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, model.ConsignmentFilter{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentsByTraderID_CountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewConsignmentService(db, nil, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnError(errors.New("count error"))

	limit := 10
	offset := 0
	result, err := svc.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, model.ConsignmentFilter{})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
