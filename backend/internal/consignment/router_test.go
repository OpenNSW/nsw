package consignment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/OpenNSW/nsw/backend/internal/auth"
	"github.com/OpenNSW/nsw/backend/internal/profile/cha"
	"github.com/OpenNSW/nsw/backend/internal/profile/company"
	"github.com/OpenNSW/nsw/backend/internal/profile/user"
)

func withAuthContext(ctx context.Context, userID string) context.Context {
	authCtx := &auth.AuthContext{
		User: &auth.UserContext{
			ID:    userID,
			Email: userID + "@example.com",
		},
	}
	return context.WithValue(ctx, auth.AuthContextKey, authCtx)
}

func withAuthContextOU(ctx context.Context, userID, ouHandle string) context.Context {
	authCtx := &auth.AuthContext{
		User: &auth.UserContext{
			ID:       userID,
			Email:    userID + "@example.com",
			OUHandle: ouHandle,
		},
	}
	return context.WithValue(ctx, auth.AuthContextKey, authCtx)
}

func TestConsignmentRouter_HandleGetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	svc := NewService(db, nil, nil, nil, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))
	r := NewRouter(svc, nil, nil, nil)

	consignmentID := uuid.NewString()
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(consignmentID, "IN_PROGRESS"))

	mockWM.On("GetStatus", mock.Anything, consignmentID).Return((*workflowManagerV2.WorkflowInstance)(nil), nil)

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+consignmentID, nil)
	req.SetPathValue("id", consignmentID)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, nil, mockCompany, nil, nil)
	r := NewRouter(svc, nil, mockCompany, nil)

	traderID := "trader1"
	companyID := "company-trader"
	mockCompany.On("GetCompanyByOUHandle", mock.Anything, "trader-ou").Return(&company.Record{ID: companyID, OUHandle: "trader-ou"}, nil)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "trader_company_id"}).AddRow(uuid.NewString(), traderID, companyID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"workflow_id", "total", "completed"}).AddRow(uuid.NewString(), 1, 0))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflows\"").WillReturnRows(sqlmock.NewRows([]string{"id", "end_node_id"}))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=trader&state=IN_PROGRESS&flow=IMPORT", nil)
	req = req.WithContext(withAuthContextOU(req.Context(), traderID, "trader-ou"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	mockCompany.AssertExpectations(t)
}

func TestConsignmentRouter_HandleGetConsignments_RoleCHA(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, nil, mockCompany, nil, nil)
	r := NewRouter(svc, nil, mockCompany, nil)

	companyID := "company-cha"
	mockCompany.On("GetCompanyByOUHandle", mock.Anything, "cha-ou").Return(&company.Record{ID: companyID, OUHandle: "cha-ou", HasCHA: true}, nil)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=cha", nil)
	req = req.WithContext(withAuthContextOU(req.Context(), "cha", "cha-ou"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	mockCompany.AssertExpectations(t)
}

func TestConsignmentRouter_HandleCreateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	mockUser := new(MockUserService)
	svc := NewService(db, nil, nil, mockCompany, mockUser, nil)
	r := NewRouter(svc, nil, nil, nil)

	traderID := "trader1"
	chaCompanyID := "cha-company"
	traderCompanyID := "trader-company"
	consignmentID := uuid.NewString()

	mockCompany.On("GetCompanyByID", mock.Anything, chaCompanyID).Return(&company.Record{ID: chaCompanyID, HasCHA: true}, nil)
	mockUser.On("GetUser", traderID).Return(&user.Record{ID: traderID, OUHandle: "trader-ou"}, nil)
	mockCompany.On("GetCompanyByOUHandle", mock.Anything, "trader-ou").Return(&company.Record{ID: traderCompanyID}, nil)

	payload := CreateConsignmentDTO{
		Flow:         FlowImport,
		ChaCompanyID: chaCompanyID,
	}
	body, _ := json.Marshal(payload)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"consignments\"").
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(
		sqlmock.NewRows([]string{"id", "flow", "trader_id", "trader_company_id", "cha_company_id", "state", "items"}).
			AddRow(consignmentID, string(FlowImport), traderID, traderCompanyID, chaCompanyID, string(Initialized), []byte("[]")),
	)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContextOU(req.Context(), traderID, "trader-ou"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	mockCompany.AssertExpectations(t)
	mockUser.AssertExpectations(t)
}

func TestConsignmentRouter_HandleInitializeConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	mockCHA := new(MockCHAService)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, mockCHA, mockCompany, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))
	r := NewRouter(svc, mockCHA, nil, nil)

	id := uuid.NewString()
	hsID := uuid.NewString()
	wtID := uuid.NewString()
	chaID := "cha-1"
	chaCompanyID := "cha-company"
	traderCompanyID := "trader-company"
	chaEmail := "cha1@example.com"

	mockCHA.On("GetByEmail", mock.Anything, chaEmail).Return(&cha.Record{ID: chaID, CompanyID: chaCompanyID}, nil)
	mockCHA.On("GetByID", mock.Anything, chaID).Return(&cha.Record{ID: chaID, CompanyID: chaCompanyID}, nil)
	mockCompany.On("GetCompanyByID", mock.Anything, traderCompanyID).Return(&company.Record{ID: traderCompanyID, OUHandle: "trader-ou", Data: []byte(`{}`)}, nil)

	payload := InitializeConsignmentDTO{HSCodeIDs: []string{hsID}}
	body, _ := json.Marshal(payload)

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow", "cha_company_id", "trader_company_id"}).AddRow(id, "INITIALIZED", "IMPORT", chaCompanyID, traderCompanyID))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)UPDATE \"consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_template_map\"").
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code_id", "consignment_flow", "workflow_template_id"}).
			AddRow(uuid.NewString(), hsID, "IMPORT", wtID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_template_v2\"").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "version", "workflow_definition"}).
			AddRow(wtID, "tmpl", "v1", []byte(`{"id":"template1"}`)))
	mockWM.On("StartWorkflow", mock.Anything, id, workflowManagerV2.WorkflowDefinition{ID: "template1"}, mock.Anything).Return(nil)
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state", "items", "created_at", "updated_at"}).AddRow(id, "IN_PROGRESS", []byte("[]"), time.Now(), time.Now()))
	mockWM.On("GetStatus", mock.Anything, id).Return(&workflowManagerV2.WorkflowInstance{ID: id}, nil)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	req, _ := http.NewRequest("PUT", "/api/v1/consignments/"+id, bytes.NewBuffer(body))
	req.SetPathValue("id", id)
	req = req.WithContext(withAuthContext(req.Context(), "cha1"))

	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	mockCHA.AssertExpectations(t)
	mockCompany.AssertExpectations(t)
}

func TestConsignmentRouter_HandleInitializeConsignment_NoID(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	req, _ := http.NewRequest("PUT", "/api/v1/consignments/", bytes.NewReader([]byte{}))
	req = req.WithContext(withAuthContext(req.Context(), "user1"))

	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment_InvalidBody(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	req, _ := http.NewRequest("PUT", "/api/v1/consignments/id", bytes.NewBufferString("invalid json"))
	req.SetPathValue("id", "id")
	req = req.WithContext(withAuthContext(req.Context(), "user1"))

	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_InvalidID(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments/invalid-uuid", nil)
	req.SetPathValue("id", "invalid-uuid")
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_PaginationError(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?limit=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_ServiceError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	id := uuid.NewString()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+id, nil)
	req.SetPathValue("id", id)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_ServiceError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCompany := new(MockCompanyService)
	svc := NewService(db, nil, nil, mockCompany, nil, nil)
	r := NewRouter(svc, nil, mockCompany, nil)
	mockCompany.On("GetCompanyByOUHandle", mock.Anything, "trader-ou").Return(&company.Record{ID: "trader-company", OUHandle: "trader-ou"}, nil)
	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	req = req.WithContext(withAuthContextOU(req.Context(), "trader1", "trader-ou"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_CompanyNotFound(t *testing.T) {
	mockCompany := new(MockCompanyService)
	svc := NewService(nil, nil, nil, mockCompany, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	mockCompany.On("GetCompanyByID", mock.Anything, "company-missing").Return(nil, company.ErrCompanyNotFound)

	payload := CreateConsignmentDTO{Flow: FlowImport, ChaCompanyID: "company-missing"}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContextOU(req.Context(), "trader1", "trader-ou"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	mockCompany.AssertExpectations(t)
}

func TestConsignmentRouter_HandleCreateConsignment_InvalidPayload(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString("invalid json"))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_InvalidRole(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil)
	r := NewRouter(svc, nil, nil, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "user1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_CompanyNotFound(t *testing.T) {
	mockCompany := new(MockCompanyService)
	svc := NewService(nil, nil, nil, mockCompany, nil, nil)
	r := NewRouter(svc, nil, mockCompany, nil)
	mockCompany.On("GetCompanyByOUHandle", mock.Anything, "cha-ou").Return(nil, company.ErrCompanyNotFound)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=cha", nil)
	req = req.WithContext(withAuthContextOU(req.Context(), "cha", "cha-ou"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil, nil, nil)
	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil, nil, nil)
	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil, nil, nil)
	req, _ := http.NewRequest("GET", "/api/v1/consignments/id", nil)
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_MissingID(t *testing.T) {
	r := NewRouter(nil, nil, nil, nil)
	req, _ := http.NewRequest("GET", "/api/v1/consignments/", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil, nil, nil)
	req, _ := http.NewRequest("PUT", "/api/v1/consignments/id", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment_EmptyHSCodes(t *testing.T) {
	r := NewRouter(nil, nil, nil, nil)
	body, _ := json.Marshal(InitializeConsignmentDTO{HSCodeIDs: []string{}})
	req, _ := http.NewRequest("PUT", "/api/v1/consignments/id", bytes.NewBuffer(body))
	req.SetPathValue("id", "id")
	req = req.WithContext(withAuthContext(req.Context(), "cha1"))
	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_CompanyLookupError(t *testing.T) {
	mockCompany := new(MockCompanyService)
	svc := NewService(nil, nil, nil, mockCompany, nil, nil)
	r := NewRouter(svc, nil, mockCompany, nil)
	mockCompany.On("GetCompanyByOUHandle", mock.Anything, "cha-ou").Return(nil, fmt.Errorf("db down"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=cha", nil)
	req = req.WithContext(withAuthContextOU(req.Context(), "cha", "cha-ou"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
