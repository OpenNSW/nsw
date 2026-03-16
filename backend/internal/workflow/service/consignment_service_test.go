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

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	workflowmanager "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// MockTemplateProvider implements TemplateProvider for testing.
type MockTemplateProvider struct {
	mock.Mock
}

func (m *MockTemplateProvider) GetWorkflowTemplateMapByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplateMap, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplateMap), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetGoWorkflowTemplateByID(ctx context.Context, id uuid.UUID) (*model.GoWorkflowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GoWorkflowTemplate), args.Error(1)
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

// MockWorkflowManager implements manager.Manager for testing.
type MockWorkflowManager struct {
	mock.Mock
}

func (m *MockWorkflowManager) StartWorkflowInstance(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID, workflowTemplates []model.WorkflowTemplate, goWorkflowTemplate *model.GoWorkflowTemplate, globalContext map[string]any, handler workflowmanager.WorkflowEventHandler) error {
	args := m.Called(ctx, tx, workflowID, workflowTemplates, goWorkflowTemplate, globalContext, handler)
	return args.Error(0)
}

func (m *MockWorkflowManager) GetWorkflowInstance(ctx context.Context, workflowID uuid.UUID) (*model.Workflow, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Workflow), args.Error(1)
}

func (m *MockWorkflowManager) RegisterTaskHandler(_ workflowmanager.TaskInitHandler) error {
	return nil
}

func (m *MockWorkflowManager) HandleTaskUpdate(_ context.Context, _ taskManager.WorkflowManagerNotification) error {
	return nil
}

func TestConsignmentService_InitializeConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewConsignmentService(db, mockTP, mockWM)

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

	workflowTemplate := &model.WorkflowTemplate{
		BaseModel:     model.BaseModel{ID: uuid.New()},
		Name:          "Test Template",
		NodeTemplates: model.UUIDArray{uuid.New()},
	}
	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlow", ctx, hsCodeID, model.ConsignmentFlowImport).Return(workflowTemplate, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockWM.On("StartWorkflowInstance", ctx, mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, (*model.GoWorkflowTemplate)(nil), globalContext, mock.Anything).Return(nil)
	sqlMock.ExpectCommit()

	nodeTemplateID := workflowTemplate.NodeTemplates[0]
	mockWM.On("GetWorkflowInstance", ctx, mock.AnythingOfType("uuid.UUID")).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: uuid.New()},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.New()},
				WorkflowNodeTemplateID: nodeTemplateID,
				State:                  model.WorkflowNodeStateReady,
				WorkflowNodeTemplate: model.WorkflowNodeTemplate{
					BaseModel: model.BaseModel{ID: nodeTemplateID},
					Name:      "Test Node",
					Type:      "SIMPLE_FORM",
				},
			},
		},
	}, nil)

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(uuid.New(), "IMPORT", traderID, "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	resp, err := svc.InitializeConsignment(ctx, createReq, traderID, globalContext)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockTP.AssertExpectations(t)
	mockWM.AssertExpectations(t)
}

func TestConsignmentService_InitializeConsignment_TemplateNotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewConsignmentService(db, mockTP, mockWM)

	ctx := context.Background()
	hsCodeID := uuid.New()
	createReq := &model.CreateConsignmentDTO{
		Flow: model.ConsignmentFlowImport,
		Items: []model.CreateConsignmentItemDTO{
			{HSCodeID: hsCodeID},
		},
	}

	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlow", ctx, hsCodeID, model.ConsignmentFlowImport).Return(nil, errors.New("template not found"))

	resp, err := svc.InitializeConsignment(ctx, createReq, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow template")
	assert.Nil(t, resp)
}

func TestConsignmentService_GetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	svc := NewConsignmentService(db, nil, mockWM)

	ctx := context.Background()
	consignmentID := uuid.New()
	hsCodeID := uuid.New()
	nodeTemplateID := uuid.New()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	mockWM.On("GetWorkflowInstance", ctx, consignmentID).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: consignmentID},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.New()},
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
	svc := NewConsignmentService(db, nil, mockWM)

	ctx := context.Background()
	consignmentID := uuid.New()
	hsCodeID := uuid.New()
	nodeTemplateID := uuid.New()

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
			AddRow(consignmentID, "IMPORT", "trader1", "FINISHED", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID.String()+`"}]`)))

	mockWM.On("GetWorkflowInstance", ctx, consignmentID).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: consignmentID},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.New()},
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
	svc := NewConsignmentService(db, nil, nil)
	ctx := context.Background()
	consignmentID := uuid.New()
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
	svc := NewConsignmentService(db, nil, nil)
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
	svc := NewConsignmentService(db, nil, nil)
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
