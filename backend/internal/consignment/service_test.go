package consignment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"
	"github.com/OpenNSW/nsw/internal/hscode"
	"github.com/OpenNSW/nsw/internal/profile/cha"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// MockTemplateProvider implements service.TemplateProvider for testing.
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

func (m *MockTemplateProvider) GetWorkflowTemplateByIDV2(ctx context.Context, id string) (*model.WorkflowTemplateV2, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplateV2), args.Error(1)
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

// MockCHAService implements cha.Service for testing.
type MockCHAService struct {
	mock.Mock
}

func (m *MockCHAService) GetByID(ctx context.Context, id string) (*cha.Record, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cha.Record), args.Error(1)
}

func (m *MockCHAService) GetByEmail(ctx context.Context, email string) (*cha.Record, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cha.Record), args.Error(1)
}

func (m *MockCHAService) List(ctx context.Context) ([]cha.Record, error) {
	args := m.Called(ctx)
	return args.Get(0).([]cha.Record), args.Error(1)
}

func (m *MockCHAService) Health() error {
	return m.Called().Error(0)
}

func TestConsignmentService_RegisterWorkflowManager(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	mockWM := new(MockWMV2)

	// Test registration
	err := svc.RegisterWorkflowManager(mockWM)
	assert.NoError(t, err)

	// Test already registered
	err = svc.RegisterWorkflowManager(mockWM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test nil manager
	svc2 := NewService(db, nil, nil, nil)
	err = svc2.RegisterWorkflowManager(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestConsignmentService_CompletionHandler(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	consignmentID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err := svc.CompletionHandler(consignmentID, nil)
	assert.NoError(t, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

// --- InitializeConsignmentByID ---

func TestConsignmentService_InitializeConsignmentByID_NoHSCode(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	_, err := svc.InitializeConsignmentByID(context.Background(), "id", []string{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one HS code ID is required")
}

func TestConsignmentService_InitializeConsignmentByID_NotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{"hs1"}, nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestConsignmentService_InitializeConsignmentByID_WrongState(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{"hs1"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be in INITIALIZED")
}

func TestConsignmentService_InitializeConsignmentByID_MultipleHSCodeError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "INITIALIZED"))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{"hs1", "hs2"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supports only one HS code")
}

func TestConsignmentService_InitializeConsignmentByID_NoTemplate(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	svc := NewService(db, mockTP, nil, nil)
	id := uuid.NewString()
	hsID := "hs1"
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow"}).AddRow(id, "INITIALIZED", "IMPORT"))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlowV2", mock.Anything, hsID, model.ConsignmentFlow("IMPORT")).Return(nil, nil)

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow template found")
}

func TestConsignmentService_InitializeConsignmentByID_Success(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWMV2)
	mockHS := hscode.NewService(db)
	svc := NewService(db, mockTP, nil, mockHS)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	id := uuid.NewString()
	hsID := uuid.NewString()
	traderID := "trader1"
	chaID := "cha1"

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "trader_id", "cha_id"}).AddRow(id, "INITIALIZED", "IMPORT", traderID, chaID))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	wt := &model.WorkflowTemplateV2{
		WorkflowDefinition: workflowManagerV2.WorkflowDefinition{ID: "template1"},
	}
	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlowV2", mock.Anything, hsID, model.ConsignmentFlow("IMPORT")).Return(wt, nil)
	mockWM.On("StartWorkflow", mock.Anything, id, wt.WorkflowDefinition, mock.Anything).Return(nil)

	sqlMock.ExpectCommit()

	// Reload (consignment.ID is populated, so GORM adds an extra "consignments"."id" = $2 clause)
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "trader_id", "cha_id", "items", "created_at", "updated_at"}).
			AddRow(id, "IN_PROGRESS", "IMPORT", traderID, chaID, []byte(`[{"hsCodeId":"`+hsID+`"}]`), time.Now(), time.Now()))

	mockWM.On("GetStatus", mock.Anything, id).Return(&workflowManagerV2.WorkflowInstance{
		ID: id,
		NodeInfo: map[string]*workflowManagerV2.NodeInfo{
			"node1": {ID: "node1", Type: workflowManagerV2.NodeTypeTask, TaskTemplateID: "tt1", Status: workflowManagerV2.NodeStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			"node2": {ID: "node2", Type: workflowManagerV2.NodeTypeEnd, Status: workflowManagerV2.NodeStatusNotStarted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
		Edges: []workflowManagerV2.Edge{
			{ID: "edge1", SourceID: "node1", TargetID: "node2"},
		},
	}, nil)

	mockTP.On("GetWorkflowNodeTemplatesByIDs", mock.Anything, []string{"tt1"}).Return([]model.WorkflowNodeTemplate{
		{BaseModel: model.BaseModel{ID: "tt1"}, Name: "Task 1", Description: "Desc 1", Type: "FORM"},
	}, nil)

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsID, "1234.56", "Test", "Cat"))

	result, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
	assert.Len(t, result.WorkflowNodes, 2)
	assert.Equal(t, "Task 1", result.WorkflowNodes[0].WorkflowNodeTemplate.Name)
	assert.Equal(t, model.WorkflowNodeStateCompleted, result.WorkflowNodes[0].State)
}

func TestConsignmentService_OnWorkflowStatusChanged(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	id := uuid.NewString()

	// Completed
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err := svc.OnWorkflowStatusChanged(context.Background(), db, id, model.WorkflowStatusInProgress, model.WorkflowStatusCompleted, nil)
	assert.NoError(t, err)

	// Other status
	err = svc.OnWorkflowStatusChanged(context.Background(), db, id, model.WorkflowStatusInProgress, model.WorkflowStatusFailed, nil)
	assert.NoError(t, err)

	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
func TestConsignmentService_CreateConsignmentShell_Success(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	svc := NewService(db, nil, mockCHA, hscode.NewService(db))
	ctx := context.Background()
	chaID := uuid.NewString()
	consignmentID := uuid.NewString()
	traderID := "trader1"

	mockCHA.On("GetByID", ctx, chaID).Return(&cha.Record{ID: chaID, Name: "Test CHA"}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), string(FlowImport), traderID, string(Initialized), sqlmock.AnyArg(), chaID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "cha_id", "state", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, chaID, "INITIALIZED", []byte("[]")))

	result, err := svc.CreateConsignmentShell(ctx, FlowImport, chaID, traderID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	mockCHA.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	svc := NewService(db, nil, nil, hscode.NewService(db))
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	ctx := context.Background()
	consignmentID := uuid.NewString()
	hsCodeID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID+`"}]`)))

	mockWM.On("GetStatus", ctx, consignmentID).Return((*workflowManagerV2.WorkflowInstance)(nil), nil)

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	result, err := svc.GetConsignmentByID(ctx, consignmentID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	assert.Len(t, result.WorkflowNodes, 0)
	mockWM.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentsByTraderID_Empty(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, hscode.NewService(db))
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	limit := 10
	offset := 0
	result, err := svc.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, Filter{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentsByTraderID_CountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, hscode.NewService(db))
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnError(errors.New("count error"))

	limit := 10
	offset := 0
	result, err := svc.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, Filter{})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_ListConsignments_NoIdentity(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	_, err := svc.ListConsignments(context.Background(), Filter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TraderID or ChaID must be set")
}

func TestConsignmentService_CreateConsignmentShell_CHANotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	mockCHA := new(MockCHAService)
	svc := NewService(db, nil, mockCHA, nil)
	ctx := context.Background()
	mockCHA.On("GetByID", ctx, "missing").Return(nil, cha.ErrCHANotFound)

	_, err := svc.CreateConsignmentShell(ctx, FlowImport, "missing", "trader1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CHA not found")
}

func TestConsignmentService_CreateConsignmentShell_InsertError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	svc := NewService(db, nil, mockCHA, nil)
	ctx := context.Background()
	chaID := uuid.NewString()
	mockCHA.On("GetByID", ctx, chaID).Return(&cha.Record{ID: chaID}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).WillReturnError(errors.New("insert failed"))
	sqlMock.ExpectRollback()

	_, err := svc.CreateConsignmentShell(ctx, FlowImport, chaID, "trader1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create consignment")
}

func TestConsignmentService_InitializeConsignmentByID_StartWorkflowError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWMV2)
	svc := NewService(db, mockTP, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	id := uuid.NewString()
	hsID := "hs1"
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow"}).AddRow(id, "INITIALIZED", "IMPORT"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	wt := &model.WorkflowTemplateV2{WorkflowDefinition: workflowManagerV2.WorkflowDefinition{ID: "tmpl"}}
	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlowV2", mock.Anything, hsID, model.ConsignmentFlow("IMPORT")).Return(wt, nil)
	mockWM.On("StartWorkflow", mock.Anything, id, wt.WorkflowDefinition, mock.Anything).Return(errors.New("start failed"))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register workflow")
}

func TestConsignmentService_InitializeConsignmentByID_TemplateProviderError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	svc := NewService(db, mockTP, nil, nil)

	id := uuid.NewString()
	hsID := "hs1"
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow"}).AddRow(id, "INITIALIZED", "IMPORT"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlowV2", mock.Anything, hsID, model.ConsignmentFlow("IMPORT")).
		Return(nil, errors.New("provider error"))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow template")
}

func TestConsignmentService_MarkConsignmentAsFinished_NotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	err := svc.OnWorkflowStatusChanged(context.Background(), db, id, model.WorkflowStatusInProgress, model.WorkflowStatusCompleted, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve consignment")
}

func TestConsignmentService_GetConsignmentByID_WMError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	svc := NewService(db, nil, nil, hscode.NewService(db))
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))
	mockWM.On("GetStatus", mock.Anything, id).Return((*workflowManagerV2.WorkflowInstance)(nil), errors.New("wm down"))

	_, err := svc.GetConsignmentByID(context.Background(), id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow details")
}

func TestConsignmentService_GetConsignmentByID_Initialized_SkipsWM(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	// No WM registered — INITIALIZED path must NOT call it.
	svc := NewService(db, nil, nil, hscode.NewService(db))

	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "items"}).AddRow(id, "INITIALIZED", []byte("[]")))

	result, err := svc.GetConsignmentByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, Initialized, result.State)
}

func TestConsignmentService_ListConsignments_WithItems(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, hscode.NewService(db))
	traderID := "trader1"
	consignmentID := uuid.NewString()
	hsID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "items", "created_at", "updated_at"}).
			AddRow(consignmentID, "IMPORT", traderID, "IN_PROGRESS", []byte(`[{"hsCodeId":"`+hsID+`"}]`), time.Now(), time.Now()))
	sqlMock.ExpectQuery(`SELECT workflow_id`).
		WillReturnRows(sqlmock.NewRows([]string{"workflow_id", "total", "completed"}).AddRow(consignmentID, 3, 1))
	endNodeID := "end1"
	sqlMock.ExpectQuery(`SELECT id, end_node_id FROM "workflows"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "end_node_id"}).AddRow(consignmentID, &endNodeID))
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsID, "1234.56", "Test", "Cat"))

	result, err := svc.ListConsignments(context.Background(), Filter{TraderID: &traderID})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.TotalCount)
	require.Len(t, result.Items, 1)
	// End node subtracted: total was 3, becomes 2.
	assert.Equal(t, 2, result.Items[0].WorkflowNodeCount)
	assert.Equal(t, 1, result.Items[0].CompletedWorkflowNodeCount)
}

func TestConsignmentService_ListConsignments_FindError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnError(errors.New("find error"))

	_, err := svc.ListConsignments(context.Background(), Filter{TraderID: &traderID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve consignments")
}

func TestConsignmentService_ListConsignments_NodeCountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "items"}).AddRow(uuid.NewString(), []byte("[]")))
	sqlMock.ExpectQuery(`SELECT workflow_id`).WillReturnError(errors.New("node count error"))

	_, err := svc.ListConsignments(context.Background(), Filter{TraderID: &traderID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node counts")
}

func TestConsignmentService_ListConsignments_EndNodeError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "items"}).AddRow(uuid.NewString(), []byte("[]")))
	sqlMock.ExpectQuery(`SELECT workflow_id`).
		WillReturnRows(sqlmock.NewRows([]string{"workflow_id", "total", "completed"}))
	sqlMock.ExpectQuery(`SELECT id, end_node_id FROM "workflows"`).
		WillReturnError(errors.New("end node error"))

	_, err := svc.ListConsignments(context.Background(), Filter{TraderID: &traderID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end nodes")
}

func TestConsignmentService_ListConsignments_ChaIDPath(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	chaID := "cha1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE cha_id = \$1`).
		WithArgs(chaID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	result, err := svc.ListConsignments(context.Background(), Filter{ChaID: &chaID})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), result.TotalCount)
}

func TestConsignmentService_BuildItemResponseDTOs_MissingHSCode(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	_, err := svc.buildConsignmentItemResponseDTOs([]Item{{HSCodeID: "missing"}}, map[string]hscode.HSCode{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HS code not found")
}
