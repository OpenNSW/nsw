package consignment

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"
	"github.com/OpenNSW/nsw/backend/internal/workflow/model"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

// MockTaskStore implements consignment.TaskStore for testing.
type MockTaskStore struct {
	mock.Mock
}

func (m *MockTaskStore) GetAllTasks(ctx context.Context, parentWorkflowID string) []tfstore.TaskRecord {
	args := m.Called(ctx, parentWorkflowID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]tfstore.TaskRecord)
}

// MockWMV2 implements workflowManagerV2.TemporalManager for testing.
type MockWMV2 struct {
	mock.Mock
}

func (m *MockWMV2) StartWorkflow(ctx context.Context, ID string, workflowDefinition workflowManagerV2.WorkflowDefinition, initialWorkflowVariables map[string]any) error {
	args := m.Called(ctx, ID, workflowDefinition, initialWorkflowVariables)
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

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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
