package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/OpenNSW/nsw/internal/auth"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	workflowManagerV1 "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

type MockTemplateProvider struct {
	mock.Mock
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, id string, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, id, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlowV2(ctx context.Context, id string, flow model.ConsignmentFlow) (*model.WorkflowTemplateV2, error) {
	args := m.Called(ctx, id, flow)
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

func setupRouterTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a gorm database connection", err)
	}

	return db, sqlMock
}

func withAuthContext(ctx context.Context, userID string) context.Context {
	authCtx := &auth.AuthContext{
		UserID: &userID,
		UserContext: &auth.UserContext{
			UserID:      userID,
			UserContext: json.RawMessage(`{}`),
		},
	}
	return context.WithValue(ctx, auth.AuthContextKey, authCtx)
}

func TestConsignmentRouter_HandleGetConsignmentByID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	mockWM := new(MockWorkflowManager)
	// TODO: Add tests for workflow manager v2
	// MockWMV2 := new(MockWMV2)
	svc := service.NewConsignmentService(db, nil, mockWM, nil)
	r := NewConsignmentRouter(svc, nil)

	consignmentID := uuid.NewString()
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	mockWM.On("GetWorkflowInstance", mock.Anything, consignmentID).Return(&model.Workflow{
		BaseModel:     model.BaseModel{ID: consignmentID},
		Status:        model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{},
	}, nil)

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+consignmentID, nil)
	req.SetPathValue("id", consignmentID)

	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	traderID := "trader1"
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id"}).AddRow(uuid.NewString(), traderID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"workflow_id", "total", "completed"}).AddRow(uuid.NewString(), 1, 0))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflows\"").WillReturnRows(sqlmock.NewRows([]string{"id", "end_node_id"}))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=trader", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	traderID := "trader1"
	chaID := uuid.NewString()
	consignmentID := uuid.NewString()

	payload := model.CreateConsignmentDTO{
		Flow:  model.ConsignmentFlowImport,
		ChaID: chaID,
	}
	body, _ := json.Marshal(payload)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"customs_house_agents\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "email"}).AddRow(chaID, "Test CHA", "", "cha@example.com"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(
		sqlmock.NewRows([]string{"id", "flow", "trader_id", "cha_id", "state", "items"}).
			AddRow(consignmentID, string(model.ConsignmentFlowImport), traderID, chaID, string(model.ConsignmentStateInitialized), []byte("[]")),
	)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewHSCodeService(db)
	r := NewHSCodeRouter(svc)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(uuid.NewString(), "1234.56"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := httptest.NewRecorder()
	r.HandleGetAllHSCodes(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreConsignmentRouter_HandleGetPreConsignmentByID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	mockWM := new(MockWorkflowManager)
	svc := service.NewPreConsignmentService(db, nil, mockWM)
	r := NewPreConsignmentRouter(svc)

	id := uuid.NewString()
	templateID := uuid.NewString()
	nodeTemplateID := uuid.NewString()

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(id, templateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))

	mockWM.On("GetWorkflowInstance", mock.Anything, id).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: id},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.NewString()},
				WorkflowNodeTemplateID: nodeTemplateID,
				State:                  model.WorkflowNodeStateReady,
				WorkflowNodeTemplate: model.WorkflowNodeTemplate{
					BaseModel: model.BaseModel{ID: nodeTemplateID},
					Type:      "TEST",
				},
			},
		},
	}, nil)

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/"+id, nil)
	req.SetPathValue("preConsignmentId", id)
	w := httptest.NewRecorder()
	r.HandleGetPreConsignmentByID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreConsignmentRouter_HandleGetTraderPreConsignments(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	traderID := "trader1"
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uuid.NewString(), "Template"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM .pre_consignments.").WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "pre_consignment_template_id"}).AddRow(uuid.NewString(), traderID, uuid.NewString()))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM .pre_consignment_templates.").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/templates", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleGetTraderPreConsignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreConsignmentRouter_HandleCreatePreConsignment(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	tp := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := service.NewPreConsignmentService(db, tp, mockWM)
	r := NewPreConsignmentRouter(svc)

	traderID := "trader1"
	templateID := uuid.NewString()
	nodeTemplateID := "00000000-0000-0000-0000-000000000001"
	preConsignmentID := uuid.NewString()

	payload := model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}
	body, _ := json.Marshal(payload)

	tp.On("GetWorkflowTemplateByID", mock.Anything, mock.Anything).Return(&model.WorkflowTemplate{BaseModel: model.BaseModel{ID: templateID}, NodeTemplates: []string{nodeTemplateID}}, nil)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).AddRow(templateID, uuid.NewString(), []byte("[]")))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"pre_consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))

	mockWM.On("StartWorkflowInstance", mock.Anything, mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Return(nil)

	sqlMock.ExpectCommit()

	// Post-commit reloads
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(preConsignmentID, templateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))

	mockWM.On("GetWorkflowInstance", mock.Anything, mock.AnythingOfType("string")).Return(&model.Workflow{
		BaseModel: model.BaseModel{ID: preConsignmentID},
		Status:    model.WorkflowStatusInProgress,
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.NewString()},
				WorkflowNodeTemplateID: nodeTemplateID,
				State:                  model.WorkflowNodeStateReady,
				WorkflowNodeTemplate: model.WorkflowNodeTemplate{
					BaseModel: model.BaseModel{ID: nodeTemplateID},
					Type:      "TEST",
				},
			},
		},
	}, nil)

	req, _ := http.NewRequest("POST", "/api/v1/pre-consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleCreatePreConsignment(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPreConsignmentRouter_HandleCreatePreConsignment_InvalidPayload(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	req, _ := http.NewRequest("POST", "/api/v1/pre-consignments", bytes.NewBufferString("invalid json"))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreatePreConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_InvalidID(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments/invalid-uuid", nil)
	req.SetPathValue("id", "invalid-uuid")
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_PaginationError(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?limit=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	id := uuid.NewString()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+id, nil)
	req.SetPathValue("id", id)

	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPreConsignmentRouter_HandleGetTraderPreConsignments_PaginationError(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/templates?limit=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetTraderPreConsignments(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewHSCodeService(db)
	r := NewHSCodeRouter(svc)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := httptest.NewRecorder()
	r.HandleGetAllHSCodes(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_InvalidPayload(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	r := NewConsignmentRouter(service.NewConsignmentService(db, nil, nil, nil), nil)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString("invalid json"))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreConsignmentRouter_HandleGetTraderPreConsignments_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewPreConsignmentService(db, nil, nil)
	r := NewPreConsignmentRouter(svc)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/templates", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetTraderPreConsignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPreConsignmentRouter_HandleGetPreConsignmentByID_InvalidID(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	r := NewPreConsignmentRouter(service.NewPreConsignmentService(db, nil, nil))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/invalid-uuid", nil)
	req.SetPathValue("preConsignmentId", "invalid-uuid")
	w := httptest.NewRecorder()
	r.HandleGetPreConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPreConsignmentRouter_HandleGetPreConsignmentByID_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	r := NewPreConsignmentRouter(service.NewPreConsignmentService(db, nil, nil))

	id := uuid.NewString()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/"+id, nil)
	req.SetPathValue("preConsignmentId", id)
	w := httptest.NewRecorder()
	r.HandleGetPreConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes_PaginationError(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	r := NewHSCodeRouter(service.NewHSCodeService(db))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes?limit=invalid", nil)
	w := httptest.NewRecorder()
	r.HandleGetAllHSCodes(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreConsignmentRouter_HandleCreatePreConsignment_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	tp := new(MockTemplateProvider)
	r := NewPreConsignmentRouter(service.NewPreConsignmentService(db, tp, nil))

	templateID := uuid.NewString()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnError(fmt.Errorf("db error"))

	payload := model.CreatePreConsignmentDTO{PreConsignmentTemplateID: templateID}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/pre-consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreatePreConsignment(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
