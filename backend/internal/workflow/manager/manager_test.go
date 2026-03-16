package manager

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

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

func (m *MockTaskManager) RegisterUpstreamCallback(_ taskManager.WorkflowUpdateHandler) {}

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

// MockNodeRepo implements service.WorkflowNodeRepository for testing
type MockNodeRepo struct {
	mock.Mock
}

func (m *MockNodeRepo) GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*model.WorkflowNode, error) {
	args := m.Called(ctx, tx, nodeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNode), args.Error(1)
}
func (m *MockNodeRepo) GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, nodeIDs)
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}
func (m *MockNodeRepo) CreateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, nodes)
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}
func (m *MockNodeRepo) UpdateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) error {
	return m.Called(ctx, tx, nodes).Error(0)
}
func (m *MockNodeRepo) GetWorkflowNodesByWorkflowIDInTx(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, workflowID)
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}
func (m *MockNodeRepo) GetWorkflowNodesByWorkflowIDsInTx(ctx context.Context, tx *gorm.DB, workflowIDs []uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, workflowIDs)
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}
func (m *MockNodeRepo) CountIncompleteNodesByWorkflowID(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tx, workflowID)
	return args.Get(0).(int64), args.Error(1)
}

// MockNodeTemplateProvider implements service.NodeTemplateProvider for testing
type MockNodeTemplateProvider struct {
	mock.Mock
}

func (m *MockNodeTemplateProvider) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]model.WorkflowNodeTemplate), args.Error(1)
}
func (m *MockNodeTemplateProvider) GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
}
func (m *MockNodeTemplateProvider) GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
}

func TestManager_HandleTaskNotification(t *testing.T) {
	db, _ := setupTestDB(t)
	mockTM := new(MockTaskManager)
	mockNodeRepo := new(MockNodeRepo)
	mockNTP := new(MockNodeTemplateProvider)

	_ = mockTM
	manager := NewManager(db, mockNodeRepo, mockNTP)

	t.Run("Node Lookup Error", func(t *testing.T) {
		taskID := uuid.New()
		pluginState := plugin.Completed

		notification := taskManager.WorkflowManagerNotification{
			TaskID:       taskID,
			UpdatedState: &pluginState,
		}

		mockNodeRepo.On("GetWorkflowNodeByIDInTx", mock.Anything, mock.Anything, taskID).Return(nil, gorm.ErrRecordNotFound).Once()

		err := manager.HandleTaskUpdate(context.Background(), notification)
		assert.Error(t, err)
		mockNodeRepo.AssertCalled(t, "GetWorkflowNodeByIDInTx", mock.Anything, mock.Anything, taskID)
	})
}
