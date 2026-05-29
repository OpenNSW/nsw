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
	"github.com/OpenNSW/nsw/backend/internal/hscode"
	"github.com/OpenNSW/nsw/backend/internal/profile/cha"
	"github.com/OpenNSW/nsw/backend/internal/profile/company"
	"github.com/OpenNSW/nsw/backend/internal/profile/user"
	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
)

// MockTemplateProvider implements service.TemplateProvider for testing.
type MockTemplateProvider struct {
	mock.Mock
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

// MockCompanyService implements company.Service for testing.
type MockCompanyService struct {
	mock.Mock
}

func (m *MockCompanyService) GetCompanyByID(ctx context.Context, id string) (*company.Record, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*company.Record), args.Error(1)
}

func (m *MockCompanyService) GetCompanyByOUHandle(ctx context.Context, ouHandle string) (*company.Record, error) {
	args := m.Called(ctx, ouHandle)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*company.Record), args.Error(1)
}

func (m *MockCompanyService) ListCompanies(ctx context.Context, filter company.ListFilter) ([]company.Record, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]company.Record), args.Error(1)
}

func (m *MockCompanyService) UpdateCompany(ctx context.Context, id string, data map[string]any) error {
	return m.Called(ctx, id, data).Error(0)
}

func (m *MockCompanyService) Health(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

// MockUserService implements user.Service for testing.
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUser(id string) (*user.Record, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Record), args.Error(1)
}

func (m *MockUserService) GetOrCreateUser(idpUserID, email, phone, ouID, ouHandle string) (*string, error) {
	args := m.Called(idpUserID, email, phone, ouID, ouHandle)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*string), args.Error(1)
}

func (m *MockUserService) UpdateUserData(id string, data []byte) error {
	return m.Called(id, data).Error(0)
}

func (m *MockUserService) Health() error {
	return m.Called().Error(0)
}

func TestConsignmentService_RegisterWorkflowManager(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	mockWM := new(MockWMV2)

	// Test registration
	err := svc.RegisterWorkflowManager(mockWM)
	assert.NoError(t, err)

	// Test already registered
	err = svc.RegisterWorkflowManager(mockWM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test nil manager
	svc2 := NewService(db, nil, nil, nil, nil, nil)
	err = svc2.RegisterWorkflowManager(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestConsignmentService_CompletionHandler(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
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
	svc := NewService(nil, nil, nil, nil, nil, nil)
	_, err := svc.InitializeConsignmentByID(context.Background(), "id", []string{}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one HS code ID is required")
}

func TestConsignmentService_InitializeConsignmentByID_NotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{"hs1"}, "")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestConsignmentService_InitializeConsignmentByID_WrongState(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{"hs1"}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be in INITIALIZED")
}

func TestConsignmentService_InitializeConsignmentByID_MultipleHSCodeError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	id := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "INITIALIZED"))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{"hs1", "hs2"}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supports only one HS code")
}

func TestConsignmentService_InitializeConsignmentByID_NoTemplate(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, mockCHA, mockCompany, nil, nil)
	id := uuid.NewString()
	hsID := "hs1"
	chaID := "cha1"
	chaCompanyID := "company-cha"
	traderCompanyID := "company-trader"

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "cha_company_id", "trader_company_id"}).AddRow(id, "INITIALIZED", "IMPORT", chaCompanyID, traderCompanyID))

	mockCHA.On("GetByID", mock.Anything, chaID).Return(&cha.Record{ID: chaID, CompanyID: chaCompanyID}, nil)
	mockCompany.On("GetCompanyByID", mock.Anything, traderCompanyID).Return(&company.Record{ID: traderCompanyID, Data: []byte(`{}`)}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_map"`).
		WithArgs(hsID, "IMPORT", 1).
		WillReturnError(gorm.ErrRecordNotFound)
	sqlMock.ExpectRollback()

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, chaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow template found")
}

func TestConsignmentService_InitializeConsignmentByID_Success(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWMV2)
	mockHS := hscode.NewService(db)
	mockCHA := new(MockCHAService)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, mockTP, mockCHA, mockCompany, nil, mockHS)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	id := uuid.NewString()
	hsID := uuid.NewString()
	traderID := "trader1"
	chaID := "cha1"
	chaCompanyID := "company-cha"
	traderCompanyID := "company-trader"

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "trader_id", "cha_company_id", "trader_company_id"}).AddRow(id, "INITIALIZED", "IMPORT", traderID, chaCompanyID, traderCompanyID))

	mockCHA.On("GetByID", mock.Anything, chaID).Return(&cha.Record{ID: chaID, CompanyID: chaCompanyID}, nil)
	mockCompany.On("GetCompanyByID", mock.Anything, traderCompanyID).Return(&company.Record{ID: traderCompanyID, OUHandle: "trader-ou", Data: []byte(`{"br_no":"BR-1"}`)}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	wtID := uuid.NewString()
	wfDef := workflowManagerV2.WorkflowDefinition{ID: "template1"}
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_map"`).
		WithArgs(hsID, "IMPORT", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code_id", "consignment_flow", "workflow_template_id"}).
			AddRow(uuid.NewString(), hsID, "IMPORT", wtID))
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_v2"`).
		WithArgs(wtID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "version", "workflow_definition"}).
			AddRow(wtID, "tmpl", "v1", []byte(`{"id":"template1"}`)))
	mockWM.On("StartWorkflow", mock.Anything, id, wfDef, mock.MatchedBy(func(vars map[string]any) bool {
		tc, ok := vars["traderCompany"].(map[string]any)
		return ok && tc["id"] == traderCompanyID
	})).Return(nil)

	sqlMock.ExpectCommit()

	// Reload (consignment.ID is populated, so GORM adds an extra "consignments"."id" = $2 clause)
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "trader_id", "cha_company_id", "trader_company_id", "cha_id", "items", "created_at", "updated_at"}).
			AddRow(id, "IN_PROGRESS", "IMPORT", traderID, chaCompanyID, traderCompanyID, chaID, []byte(`[{"hsCodeId":"`+hsID+`"}]`), time.Now(), time.Now()))

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

	result, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, chaID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
	assert.Len(t, result.WorkflowNodes, 2)
	assert.Equal(t, "Task 1", result.WorkflowNodes[0].WorkflowNodeTemplate.Name)
	assert.Equal(t, model.WorkflowNodeStateCompleted, result.WorkflowNodes[0].State)
	mockCHA.AssertExpectations(t)
	mockCompany.AssertExpectations(t)
}

func TestConsignmentService_OnWorkflowStatusChanged(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
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
	mockCompany := new(MockCompanyService)
	mockUser := new(MockUserService)
	svc := NewService(db, nil, nil, mockCompany, mockUser, hscode.NewService(db))
	ctx := context.Background()
	chaCompanyID := uuid.NewString()
	traderCompanyID := uuid.NewString()
	consignmentID := uuid.NewString()
	traderID := "trader1"

	mockCompany.On("GetCompanyByID", ctx, chaCompanyID).Return(&company.Record{ID: chaCompanyID, HasCHA: true}, nil)
	mockUser.On("GetUser", traderID).Return(&user.Record{ID: traderID, OUHandle: "trader-ou"}, nil)
	mockCompany.On("GetCompanyByOUHandle", ctx, "trader-ou").Return(&company.Record{ID: traderCompanyID}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "trader_company_id", "cha_company_id", "state", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, traderCompanyID, chaCompanyID, "INITIALIZED", []byte("[]")))

	result, err := svc.CreateConsignmentShell(ctx, FlowImport, chaCompanyID, traderID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	mockCompany.AssertExpectations(t)
	mockUser.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_CreateConsignmentShell_CompanyNotCHA(t *testing.T) {
	db, _ := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, nil, mockCompany, nil, nil)

	mockCompany.On("GetCompanyByID", mock.Anything, "company-1").Return(&company.Record{ID: "company-1", HasCHA: false}, nil)

	_, err := svc.CreateConsignmentShell(context.Background(), FlowImport, "company-1", "trader1")
	assert.ErrorIs(t, err, ErrCompanyNotCHA)
}

func TestConsignmentService_GetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	svc := NewService(db, nil, nil, nil, nil, hscode.NewService(db))
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

func TestConsignmentService_ListConsignments_TraderCompany_Empty(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, hscode.NewService(db))
	ctx := context.Background()
	companyID := "company-1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_company_id = \$1`).
		WithArgs(companyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	result, err := svc.ListConsignments(ctx, Filter{TraderCompanyID: &companyID})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_ListConsignments_TraderCompany_CountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, hscode.NewService(db))
	ctx := context.Background()
	companyID := "company-1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnError(errors.New("count error"))

	result, err := svc.ListConsignments(ctx, Filter{TraderCompanyID: &companyID})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_ListConsignments_NoIdentity(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	_, err := svc.ListConsignments(context.Background(), Filter{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TraderCompanyID or CHACompanyID must be set")
}

func TestConsignmentService_CreateConsignmentShell_CompanyNotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, nil, mockCompany, nil, nil)
	ctx := context.Background()
	mockCompany.On("GetCompanyByID", ctx, "missing").Return(nil, company.ErrCompanyNotFound)

	_, err := svc.CreateConsignmentShell(ctx, FlowImport, "missing", "trader1")
	assert.ErrorIs(t, err, company.ErrCompanyNotFound)
}

func TestConsignmentService_CreateConsignmentShell_InsertError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	mockUser := new(MockUserService)
	svc := NewService(db, nil, nil, mockCompany, mockUser, nil)
	ctx := context.Background()
	chaCompanyID := uuid.NewString()
	traderID := "trader1"
	mockCompany.On("GetCompanyByID", ctx, chaCompanyID).Return(&company.Record{ID: chaCompanyID, HasCHA: true}, nil)
	mockUser.On("GetUser", traderID).Return(&user.Record{ID: traderID, OUHandle: "trader-ou"}, nil)
	mockCompany.On("GetCompanyByOUHandle", ctx, "trader-ou").Return(&company.Record{ID: "trader-company"}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).WillReturnError(errors.New("insert failed"))
	sqlMock.ExpectRollback()

	_, err := svc.CreateConsignmentShell(ctx, FlowImport, chaCompanyID, traderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create consignment")
}

func TestConsignmentService_InitializeConsignmentByID_StartWorkflowError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	mockCHA := new(MockCHAService)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, mockCHA, mockCompany, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	id := uuid.NewString()
	hsID := "hs1"
	wtID := uuid.NewString()
	chaID := "cha1"
	chaCompanyID := "company-cha"
	traderCompanyID := "company-trader"

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "cha_company_id", "trader_company_id"}).AddRow(id, "INITIALIZED", "IMPORT", chaCompanyID, traderCompanyID))

	mockCHA.On("GetByID", mock.Anything, chaID).Return(&cha.Record{ID: chaID, CompanyID: chaCompanyID}, nil)
	mockCompany.On("GetCompanyByID", mock.Anything, traderCompanyID).Return(&company.Record{ID: traderCompanyID, Data: []byte(`{}`)}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_map"`).
		WithArgs(hsID, "IMPORT", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code_id", "consignment_flow", "workflow_template_id"}).
			AddRow(uuid.NewString(), hsID, "IMPORT", wtID))
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_v2"`).
		WithArgs(wtID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "version", "workflow_definition"}).
			AddRow(wtID, "tmpl", "v1", []byte(`{"id":"tmpl"}`)))
	mockWM.On("StartWorkflow", mock.Anything, id, workflowManagerV2.WorkflowDefinition{ID: "tmpl"}, mock.Anything).Return(errors.New("start failed"))

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, chaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register workflow")
}

func TestConsignmentService_InitializeConsignmentByID_TemplateProviderError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, mockCHA, mockCompany, nil, nil)

	id := uuid.NewString()
	hsID := "hs1"
	chaID := "cha1"
	chaCompanyID := "company-cha"
	traderCompanyID := "company-trader"

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "cha_company_id", "trader_company_id"}).AddRow(id, "INITIALIZED", "IMPORT", chaCompanyID, traderCompanyID))

	mockCHA.On("GetByID", mock.Anything, chaID).Return(&cha.Record{ID: chaID, CompanyID: chaCompanyID}, nil)
	mockCompany.On("GetCompanyByID", mock.Anything, traderCompanyID).Return(&company.Record{ID: traderCompanyID, Data: []byte(`{}`)}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_map"`).
		WithArgs(hsID, "IMPORT", 1).
		WillReturnError(errors.New("provider error"))
	sqlMock.ExpectRollback()

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, chaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow template")
}

func TestConsignmentService_InitializeConsignmentByID_CHACompanyMismatch(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, mockCHA, mockCompany, nil, nil)

	id := uuid.NewString()
	hsID := "hs1"
	chaID := "cha1"

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "cha_company_id"}).AddRow(id, "INITIALIZED", "IMPORT", "company-A"))

	mockCHA.On("GetByID", mock.Anything, chaID).Return(&cha.Record{ID: chaID, CompanyID: "company-B"}, nil)

	_, err := svc.InitializeConsignmentByID(context.Background(), id, []string{hsID}, chaID)
	assert.ErrorIs(t, err, ErrCHACompanyMismatch)
}

func TestConsignmentService_MarkConsignmentAsFinished_NotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
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
	svc := NewService(db, nil, nil, nil, nil, hscode.NewService(db))
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
	svc := NewService(db, nil, nil, nil, nil, hscode.NewService(db))

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
	svc := NewService(db, nil, nil, nil, nil, hscode.NewService(db))
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

	result, err := svc.ListConsignments(context.Background(), Filter{TraderCompanyID: &traderID})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.TotalCount)
	require.Len(t, result.Items, 1)
	// End node subtracted: total was 3, becomes 2.
	assert.Equal(t, 2, result.Items[0].WorkflowNodeCount)
	assert.Equal(t, 1, result.Items[0].CompletedWorkflowNodeCount)
}

func TestConsignmentService_ListConsignments_FindError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnError(errors.New("find error"))

	_, err := svc.ListConsignments(context.Background(), Filter{TraderCompanyID: &traderID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve consignments")
}

func TestConsignmentService_ListConsignments_NodeCountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "items"}).AddRow(uuid.NewString(), []byte("[]")))
	sqlMock.ExpectQuery(`SELECT workflow_id`).WillReturnError(errors.New("node count error"))

	_, err := svc.ListConsignments(context.Background(), Filter{TraderCompanyID: &traderID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node counts")
}

func TestConsignmentService_ListConsignments_EndNodeError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "consignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "items"}).AddRow(uuid.NewString(), []byte("[]")))
	sqlMock.ExpectQuery(`SELECT workflow_id`).
		WillReturnRows(sqlmock.NewRows([]string{"workflow_id", "total", "completed"}))
	sqlMock.ExpectQuery(`SELECT id, end_node_id FROM "workflows"`).
		WillReturnError(errors.New("end node error"))

	_, err := svc.ListConsignments(context.Background(), Filter{TraderCompanyID: &traderID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end nodes")
}

func TestConsignmentService_ListConsignments_CHACompanyPath(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	companyID := "company-cha"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE cha_company_id = \$1`).
		WithArgs(companyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	result, err := svc.ListConsignments(context.Background(), Filter{CHACompanyID: &companyID})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), result.TotalCount)
}

func TestConsignmentService_BuildItemResponseDTOs_MissingHSCode(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil)
	_, err := svc.buildConsignmentItemResponseDTOs([]Item{{HSCodeID: "missing"}}, map[string]hscode.HSCode{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HS code not found")
}
