package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/auth"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a gorm database", err)
	}

	return gormDB, mock
}

// MockTaskManager is a mock implementation of taskManager.TaskManager
type MockTaskManager struct {
	mock.Mock
}

func (m *MockTaskManager) InitTask(ctx context.Context, req taskManager.InitTaskRequest) (*taskManager.InitTaskResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*taskManager.InitTaskResponse), args.Error(1)
}

func (m *MockTaskManager) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func (m *MockTaskManager) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func TestPluginStateToWorkflowNodeState(t *testing.T) {
	tests := []struct {
		name          string
		input         plugin.State
		expectedState model.WorkflowNodeState
		expectError   bool
	}{
		{
			name:          "InProgress",
			input:         plugin.InProgress,
			expectedState: model.WorkflowNodeStateInProgress,
			expectError:   false,
		},
		{
			name:          "Completed",
			input:         plugin.Completed,
			expectedState: model.WorkflowNodeStateCompleted,
			expectError:   false,
		},
		{
			name:          "Failed",
			input:         plugin.Failed,
			expectedState: model.WorkflowNodeStateFailed,
			expectError:   false,
		},
		{
			name:          "Unknown",
			input:         plugin.State("unknown"),
			expectedState: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pluginStateToWorkflowNodeState(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedState, result)
			}
		})
	}
}

func TestManager_StartWorkflowNodeUpdateListener(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)

	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	t.Run("Node Lookup Error", func(t *testing.T) {
		taskID := uuid.New()
		pluginState := plugin.Completed

		notification := taskManager.WorkflowManagerNotification{
			TaskID:       taskID,
			UpdatedState: &pluginState,
		}

		sqlMock.ExpectQuery(".*").WillReturnError(gorm.ErrRecordNotFound)

		ch <- notification

		assert.Eventually(t, func() bool {
			return sqlMock.ExpectationsWereMet() == nil
		}, 200*time.Millisecond, 10*time.Millisecond)
	})

	manager.StopWorkflowNodeUpdateListener()
}

func TestManager_HandleGetAllHSCodes(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM .*hs_codes").WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(uuid.New(), "1234.56"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := parseHTTPResponse(t, manager.HandleGetAllHSCodes, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestManager_HandleGetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	id := uuid.New()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM .*consignments").WillReturnError(gorm.ErrRecordNotFound)

	req, _ := http.NewRequest("GET", "/api/v1/consignments/"+id.String(), nil)
	req.SetPathValue("id", id.String())

	w := parseHTTPResponse(t, manager.HandleGetConsignmentByID, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestManager_HandleGetConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	req, _ := http.NewRequest("GET", "/api/v1/consignments", nil)
	w := parseHTTPResponse(t, manager.HandleGetConsignmentsByTraderID, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestManager_HandleGetPreConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments", nil)
	w := parseHTTPResponse(t, manager.HandleGetPreConsignmentsByTraderID, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestManager_HandleGetPreConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	id := uuid.New()
	sqlMock.ExpectQuery("(?i)SELECT .* FROM .*pre_consignments").WillReturnError(gorm.ErrRecordNotFound)

	req, _ := http.NewRequest("GET", "/api/v1/pre-consignments/"+id.String(), nil)
	req.SetPathValue("preConsignmentId", id.String())

	w := parseHTTPResponse(t, manager.HandleGetPreConsignmentByID, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
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

func parseHTTPResponse(t *testing.T, handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func TestManager_HandleCreateConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	traderID := "trader1"
	hsCodeID := uuid.New()
	nodeTemplateID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	payload := model.CreateConsignmentDTO{
		Flow: model.ConsignmentFlowImport,
		Items: []model.CreateConsignmentItemDTO{
			{HSCodeID: hsCodeID},
		},
	}
	body, _ := json.Marshal(payload)

	sqlMock.MatchExpectationsInOrder(false)
	sqlMock.ExpectQuery("(?i).*template_maps").WillReturnRows(sqlmock.NewRows([]string{"id", "nodes"}).AddRow(uuid.New(), []byte(`["00000000-0000-0000-0000-000000000001"]`)))
	sqlMock.ExpectQuery("(?i).*workflow_templates").WillReturnRows(sqlmock.NewRows([]string{"id", "nodes"}).AddRow(uuid.New(), []byte(`["00000000-0000-0000-0000-000000000001"]`)))

	for i := 0; i < 5; i++ {
		sqlMock.ExpectQuery("(?i).*workflow_node_templates").WillReturnRows(sqlmock.NewRows([]string{"id", "type"}).AddRow(nodeTemplateID, "TEST_TYPE"))
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("(?i)INSERT INTO \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))

	for i := 0; i < 5; i++ {
		sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id"}).AddRow(uuid.New(), nodeTemplateID))
		sqlMock.ExpectExec("(?i)UPDATE \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))
	}

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(uuid.New(), "READY"))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

	sqlMock.ExpectCommit()

	mockTM.On("InitTask", mock.Anything, mock.Anything).Return(&taskManager.InitTaskResponse{Result: "success"}, nil)

	req, _ := http.NewRequest("POST", "/api/v1/consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), traderID))

	w := httptest.NewRecorder()
	manager.HandleCreateConsignment(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestManager_HandleCreatePreConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTM := new(MockTaskManager)
	ch := make(chan taskManager.WorkflowManagerNotification, 10)
	manager := NewManager(mockTM, ch, db)
	sqlMock.MatchExpectationsInOrder(false)

	traderID := "trader1"
	templateID := uuid.New()
	nodeTemplateID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	payload := model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}
	body, _ := json.Marshal(payload)

	sqlMock.MatchExpectationsInOrder(false)
	for i := 0; i < 3; i++ {
		sqlMock.ExpectQuery("(?i).*pre_consignment_templates").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).AddRow(templateID, uuid.New(), []byte("[]")))
		sqlMock.ExpectQuery("(?i).*workflow_templates").WillReturnRows(sqlmock.NewRows([]string{"id", "nodes"}).AddRow(uuid.New(), []byte(`["00000000-0000-0000-0000-000000000001"]`)))
		sqlMock.ExpectQuery("(?i).*workflow_node_templates").WillReturnRows(sqlmock.NewRows([]string{"id", "type"}).AddRow(nodeTemplateID, "TEST_TYPE"))
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("(?i)INSERT INTO \"pre_consignments\"").WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectExec("(?i)INSERT INTO \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))

	for i := 0; i < 5; i++ {
		sqlMock.ExpectQuery("(?i)SELECT .* FROM \"workflow_nodes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_node_template_id"}).AddRow(uuid.New(), nodeTemplateID))
		sqlMock.ExpectExec("(?i)UPDATE \"workflow_nodes\"").WillReturnResult(sqlmock.NewResult(1, 1))
	}

	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"pre_consignments\"").WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(uuid.New(), templateID))

	sqlMock.ExpectCommit()

	mockTM.On("InitTask", mock.Anything, mock.Anything).Return(&taskManager.InitTaskResponse{Result: "success"}, nil)

	req, _ := http.NewRequest("POST", "/api/v1/pre-consignments", bytes.NewBuffer(body))
	req = req.WithContext(withAuthContext(req.Context(), traderID))

	w := httptest.NewRecorder()
	manager.HandleCreatePreConsignment(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
