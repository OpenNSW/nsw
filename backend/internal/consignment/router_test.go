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
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/profile/cha"
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

func TestConsignmentRouter_HandleGetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	svc := NewService(db, nil, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))
	r := NewRouter(svc, nil)

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
	svc := NewService(db, nil, nil, nil)
	r := NewRouter(svc, nil)

	traderID := "trader1"
	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id"}).AddRow(uuid.NewString(), traderID))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"workflow_id", "total", "completed"}).AddRow(uuid.NewString(), 1, 0))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflows\"").WillReturnRows(sqlmock.NewRows([]string{"id", "end_node_id"}))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=trader&state=IN_PROGRESS&flow=IMPORT", nil)
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_RoleCHA(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	svc := NewService(db, nil, mockCHA, nil)
	r := NewRouter(svc, mockCHA)

	email := "cha@example.com"
	chaID := "cha1"
	mockCHA.On("GetByEmail", mock.Anything, email).Return(&cha.Record{ID: chaID}, nil)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=cha", nil)
	req = req.WithContext(withAuthContext(req.Context(), "cha"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, cha.NewService(db), nil)
	r := NewRouter(svc, nil)

	traderID := "trader1"
	chaID := uuid.NewString()
	consignmentID := uuid.NewString()

	payload := CreateConsignmentDTO{
		Flow:  FlowImport,
		ChaID: chaID,
	}
	body, _ := json.Marshal(payload)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"customs_house_agents\"").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "email"}).AddRow(chaID, "Test CHA", "", "cha@example.com"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"consignments\"").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), string(FlowImport), traderID, string(Initialized), sqlmock.AnyArg(), chaID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(
		sqlmock.NewRows([]string{"id", "flow", "trader_id", "cha_id", "state", "items"}).
			AddRow(consignmentID, string(FlowImport), traderID, chaID, string(Initialized), []byte("[]")),
	)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), traderID))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWMV2)
	svc := NewService(db, mockTP, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))
	r := NewRouter(svc, nil)

	id := uuid.NewString()
	hsID := uuid.NewString()
	payload := InitializeConsignmentDTO{HSCodeIDs: []string{hsID}}
	body, _ := json.Marshal(payload)

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state", "flow"}).AddRow(id, "INITIALIZED", "IMPORT"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)UPDATE \"consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))

	wt := &model.WorkflowTemplateV2{WorkflowDefinition: workflowManagerV2.WorkflowDefinition{ID: "template1"}}
	mockTP.On("GetWorkflowTemplateByHSCodeIDAndFlowV2", mock.Anything, hsID, model.ConsignmentFlow("IMPORT")).Return(wt, nil)
	mockWM.On("StartWorkflow", mock.Anything, id, wt.WorkflowDefinition, mock.Anything).Return(nil)
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
}

func TestConsignmentRouter_HandleInitializeConsignment_NoID(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	r := NewRouter(svc, nil)

	req, _ := http.NewRequest("PUT", "/api/v1/consignments/", bytes.NewReader([]byte{}))
	req = req.WithContext(withAuthContext(req.Context(), "user1"))

	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment_InvalidBody(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	r := NewRouter(svc, nil)

	req, _ := http.NewRequest("PUT", "/api/v1/consignments/id", bytes.NewBufferString("invalid json"))
	req.SetPathValue("id", "id")
	req = req.WithContext(withAuthContext(req.Context(), "user1"))

	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_InvalidID(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	r := NewRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments/invalid-uuid", nil)
	req.SetPathValue("id", "invalid-uuid")
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_PaginationError(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	r := NewRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?limit=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_ServiceError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	r := NewRouter(svc, nil)

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
	svc := NewService(db, nil, nil, nil)
	r := NewRouter(svc, nil)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_CHANotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, cha.NewService(db), nil)
	r := NewRouter(svc, nil)

	chaID := uuid.NewString()
	payload := CreateConsignmentDTO{Flow: FlowImport, ChaID: chaID}
	body, _ := json.Marshal(payload)

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"customs_house_agents\"").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_InvalidPayload(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db, nil, nil, nil)
	r := NewRouter(svc, nil)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString("invalid json"))
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_InvalidRole(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	r := NewRouter(svc, nil)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=invalid", nil)
	req = req.WithContext(withAuthContext(req.Context(), "user1"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_CHANotFound(t *testing.T) {
	mockCHA := new(MockCHAService)
	svc := NewService(nil, nil, mockCHA, nil)
	r := NewRouter(svc, mockCHA)

	mockCHA.On("GetByEmail", mock.Anything, "cha@example.com").Return(nil, cha.ErrCHANotFound)

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=cha", nil)
	req = req.WithContext(withAuthContext(req.Context(), "cha"))

	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil)
	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleCreateConsignment_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil)
	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()
	r.HandleCreateConsignment(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil)
	req, _ := http.NewRequest("GET", "/api/v1/consignments/id", nil)
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleGetConsignmentByID_MissingID(t *testing.T) {
	r := NewRouter(nil, nil)
	req, _ := http.NewRequest("GET", "/api/v1/consignments/", nil)
	req = req.WithContext(withAuthContext(req.Context(), "trader1"))
	w := httptest.NewRecorder()
	r.HandleGetConsignmentByID(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment_Unauthorized(t *testing.T) {
	r := NewRouter(nil, nil)
	req, _ := http.NewRequest("PUT", "/api/v1/consignments/id", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConsignmentRouter_HandleInitializeConsignment_EmptyHSCodes(t *testing.T) {
	r := NewRouter(nil, nil)
	body, _ := json.Marshal(InitializeConsignmentDTO{HSCodeIDs: []string{}})
	req, _ := http.NewRequest("PUT", "/api/v1/consignments/id", bytes.NewBuffer(body))
	req.SetPathValue("id", "id")
	req = req.WithContext(withAuthContext(req.Context(), "cha1"))
	w := httptest.NewRecorder()
	r.HandleInitializeConsignment(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsignmentRouter_HandleGetConsignments_CHALookupError(t *testing.T) {
	mockCHA := new(MockCHAService)
	svc := NewService(nil, nil, mockCHA, nil)
	r := NewRouter(svc, mockCHA)
	mockCHA.On("GetByEmail", mock.Anything, "cha@example.com").Return(nil, fmt.Errorf("db down"))

	req, _ := http.NewRequest("GET", "/api/v1/consignments?role=cha", nil)
	req = req.WithContext(withAuthContext(req.Context(), "cha"))
	w := httptest.NewRecorder()
	r.HandleGetConsignments(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
