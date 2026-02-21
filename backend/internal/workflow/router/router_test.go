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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

type MockTemplateProvider struct {
	mock.Mock
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, id uuid.UUID, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, id, flow)
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
	return args.Get(0).([]model.WorkflowNodeTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
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

func withAuthContext(ctx context.Context, traderID string) context.Context {
	authCtx := &auth.AuthContext{
		TraderContext: &auth.TraderContext{
			TraderID:      traderID,
			TraderContext: json.RawMessage(`{}`),
		},
	}
	return context.WithValue(ctx, auth.AuthContextKey, authCtx)
}

func TestConsignmentRouter_HandleGetConsignmentByID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	consignmentID := uuid.New()
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "consignment_id"}))

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+consignmentID.String(), nil)
	req.SetPathValue("id", consignmentID.String())

	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	traderID := "trader1"
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id"}).AddRow(uuid.New(), traderID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"consignment_id", "total", "completed"}).AddRow(uuid.New(), 1, 0))

	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleGetConsignmentsByTraderID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	tp := new(MockTemplateProvider)
	svc := service.NewConsignmentService(db, tp, nil)
	r := NewConsignmentRouter(svc, nil)

	traderID := "trader1"
	hsCodeID := uuid.New()
	templateID := uuid.New()
	nodeTemplateID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	consignmentID := uuid.New()

	payload := model.CreateConsignmentDTO{
		Flow: model.ConsignmentFlowImport,
		Items: []model.CreateConsignmentItemDTO{
			{HSCodeID: hsCodeID},
		},
	}
	body, _ := json.Marshal(payload)

	tp.On("GetWorkflowTemplateByHSCodeIDAndFlow", mock.Anything, mock.Anything, mock.Anything).Return(&model.WorkflowTemplate{BaseModel: model.BaseModel{ID: templateID}, NodeTemplates: []uuid.UUID{nodeTemplateID}}, nil)
	tp.On("GetWorkflowNodeTemplatesByIDs", mock.Anything, mock.Anything).Return([]model.WorkflowNodeTemplate{{BaseModel: model.BaseModel{ID: nodeTemplateID}, Type: "TEST"}}, nil)
	tp.On("GetWorkflowNodeTemplateByID", mock.Anything, mock.Anything).Return(&model.WorkflowNodeTemplate{BaseModel: model.BaseModel{ID: nodeTemplateID}, Type: "TEST"}, nil)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("(?i)INSERT INTO \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))

	// Queries from state machine / initialization
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id"}).AddRow(uuid.New(), nodeTemplateID))
	sqlMock.ExpectExec("(?i)UPDATE \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	// Post-commit reloads
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "READY"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "consignment_id", "workflow_node_template_id"}).AddRow(uuid.New(), consignmentID, nodeTemplateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_node_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "type"}).AddRow(nodeTemplateID, "TEST"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

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
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(uuid.New(), "1234.56"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := httptest.NewRecorder()
	r.HandleGetAllHSCodes(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreConsignmentRouter_HandleGetPreConsignmentByID(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	wr := service.NewWorkflowNodeService(db)
	svc := service.NewPreConsignmentService(db, nil, wr)
	r := NewPreConsignmentRouter(svc)

	id := uuid.New()
	templateID := uuid.New()
	nodeTemplateID := uuid.New()

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(id, templateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id", "workflow_node_template_id"}).AddRow(uuid.New(), id, nodeTemplateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_node_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "type"}).AddRow(nodeTemplateID, "TEST"))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/"+id.String(), nil)
	req.SetPathValue("preConsignmentId", id.String())
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
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(uuid.New(), "Template"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM .pre_consignments.").WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "pre_consignment_template_id"}).AddRow(uuid.New(), traderID, uuid.New()))
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
	wr := service.NewWorkflowNodeService(db)
	svc := service.NewPreConsignmentService(db, tp, wr)
	r := NewPreConsignmentRouter(svc)

	traderID := "trader1"
	templateID := uuid.New()
	nodeTemplateID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	preConsignmentID := uuid.New()

	payload := model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}
	body, _ := json.Marshal(payload)

	tp.On("GetWorkflowTemplateByID", mock.Anything, mock.Anything).Return(&model.WorkflowTemplate{BaseModel: model.BaseModel{ID: templateID}, NodeTemplates: []uuid.UUID{nodeTemplateID}}, nil)
	tp.On("GetWorkflowNodeTemplatesByIDs", mock.Anything, mock.Anything).Return([]model.WorkflowNodeTemplate{{BaseModel: model.BaseModel{ID: nodeTemplateID}, Type: "TEST"}}, nil)
	tp.On("GetWorkflowNodeTemplateByID", mock.Anything, mock.Anything).Return(&model.WorkflowNodeTemplate{BaseModel: model.BaseModel{ID: nodeTemplateID}, Type: "TEST"}, nil)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).AddRow(templateID, uuid.New(), []byte("[]")))

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"pre_consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("(?i)INSERT INTO \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))

	// State machine initialization
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id"}).AddRow(uuid.New(), nodeTemplateID))
	sqlMock.ExpectExec("(?i)UPDATE \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectCommit()

	// Post-commit reloads
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(preConsignmentID, templateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id", "workflow_node_template_id"}).AddRow(uuid.New(), preConsignmentID, nodeTemplateID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_node_templates\"").WillReturnRows(sqlmock.NewRows([]string{"id", "type"}).AddRow(nodeTemplateID, "TEST"))

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
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments/invalid-uuid", nil)
	req.SetPathValue("id", "invalid-uuid")
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentsByTraderID_PaginationError(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?limit=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignmentsByTraderID(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	id := uuid.New()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+id.String(), nil)
	req.SetPathValue("id", id.String())

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

func TestConsignmentRouter_HandleGetConsignmentsByTraderID_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	svc := service.NewConsignmentService(db, nil, nil)
	r := NewConsignmentRouter(svc, nil)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignmentsByTraderID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_InvalidPayload(t *testing.T) {
	db, _ := setupRouterTestDB(t)
	r := NewConsignmentRouter(service.NewConsignmentService(db, nil, nil), nil)

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
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreConsignmentRouter_HandleGetPreConsignmentByID_ServiceError(t *testing.T) {
	db, sqlMock := setupRouterTestDB(t)
	r := NewPreConsignmentRouter(service.NewPreConsignmentService(db, nil, nil))

	id := uuid.New()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/"+id.String(), nil)
	req.SetPathValue("preConsignmentId", id.String())
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

	templateID := uuid.New()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignment_templates\"").WillReturnError(fmt.Errorf("db error"))

	payload := model.CreatePreConsignmentDTO{PreConsignmentTemplateID: templateID}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/pre-consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreatePreConsignment(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
