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
	workflowManagerV1 "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// MockWorkflowManager implements v1 workflow manager for pre-consignment tests.
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

func TestPreConsignmentService_InitializePreConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewPreConsignmentService(db, mockTP, mockWM)

	ctx := context.Background()
	traderID := "trader1"
	templateID := uuid.NewString()
	createReq := &model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}
	initialContext := map[string]any{"key": "value"}

	// Get PreConsignmentTemplate
	workflowTemplateID := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1`).
		WithArgs(templateID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
			AddRow(templateID, workflowTemplateID, []byte("[]")))

	// Get Workflow Template
	workflowTemplate := &model.WorkflowTemplate{
		BaseModel:     model.BaseModel{ID: workflowTemplateID},
		Name:          "Test WF Template",
		NodeTemplates: model.StringArray{},
	}
	mockTP.On("GetWorkflowTemplateByID", ctx, workflowTemplateID).Return(workflowTemplate, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "pre_consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mockWM.On("StartWorkflowInstance", ctx, mock.Anything, mock.AnythingOfType("string"), mock.Anything, initialContext, mock.Anything).Return(nil)
	sqlMock.ExpectCommit()

	// Reload pre-consignment with template
	pcID := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "created_at", "updated_at", "pre_consignment_template_id"}).
			AddRow(pcID, traderID, "IN_PROGRESS", time.Now(), time.Now(), templateID))

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Test PC Template"))

	// GetWorkflowInstance for building response DTO
	nodeTemplateID := uuid.NewString()
	mockWM.On("GetWorkflowInstance", ctx, mock.AnythingOfType("string")).Return(&model.Workflow{
		BaseModel:     model.BaseModel{ID: pcID},
		Status:        model.WorkflowStatusInProgress,
		GlobalContext: map[string]any{"key": "value"},
		WorkflowNodes: []model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.NewString()},
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

	resp, err := svc.InitializePreConsignment(ctx, createReq, traderID, initialContext)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockTP.AssertExpectations(t)
	mockWM.AssertExpectations(t)
}

func TestPreConsignmentService_InitializePreConsignment_TemplateNotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewPreConsignmentService(db, mockTP, mockWM)

	ctx := context.Background()
	templateID := uuid.NewString()
	createReq := &model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1`).
		WithArgs(templateID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	resp, err := svc.InitializePreConsignment(ctx, createReq, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, resp)
}

func TestPreConsignmentService_InitializePreConsignment_WorkflowTemplateFetchError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewPreConsignmentService(db, mockTP, mockWM)

	ctx := context.Background()
	templateID := uuid.NewString()
	workflowTemplateID := uuid.NewString()
	createReq := &model.CreatePreConsignmentDTO{
		PreConsignmentTemplateID: templateID,
	}

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1`).
		WithArgs(templateID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
			AddRow(templateID, workflowTemplateID, []byte("[]")))

	mockTP.On("GetWorkflowTemplateByID", ctx, workflowTemplateID).Return(nil, errors.New("wf error"))

	resp, err := svc.InitializePreConsignment(ctx, createReq, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow template")
	assert.Nil(t, resp)
}

func TestPreConsignmentService_GetPreConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	svc := NewPreConsignmentService(db, nil, mockWM)

	ctx := context.Background()
	pcID := uuid.NewString()
	templateID := uuid.NewString()
	nodeTemplateID := uuid.NewString()

	t.Run("Success", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
				AddRow(pcID, "trader1", "IN_PROGRESS", templateID))

		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
			WithArgs(templateID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Template"))

		mockWM.On("GetWorkflowInstance", ctx, pcID).Return(&model.Workflow{
			BaseModel: model.BaseModel{ID: pcID},
			Status:    model.WorkflowStatusInProgress,
			WorkflowNodes: []model.WorkflowNode{
				{
					BaseModel:              model.BaseModel{ID: uuid.NewString()},
					WorkflowNodeTemplateID: nodeTemplateID,
					State:                  model.WorkflowNodeStateReady,
					WorkflowNodeTemplate: model.WorkflowNodeTemplate{
						BaseModel: model.BaseModel{ID: nodeTemplateID},
						Name:      "Node Template",
						Type:      "SIMPLE_FORM",
					},
				},
			},
		}, nil).Once()

		resp, err := svc.GetPreConsignmentByID(ctx, pcID)
		assert.NoError(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, pcID, resp.ID)
		}
		mockWM.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1 ORDER BY "pre_consignments"."id" LIMIT \$2`).
			WithArgs(pcID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		resp, err := svc.GetPreConsignmentByID(ctx, pcID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestPreConsignmentService_GetPreConsignmentsByTraderID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	svc := NewPreConsignmentService(db, nil, mockWM)

	ctx := context.Background()
	traderID := "trader1"

	t.Run("Success", func(t *testing.T) {
		pcID := uuid.NewString()
		templateID := uuid.NewString()
		nodeTemplateID := uuid.NewString()

		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
			WithArgs(traderID, model.PreConsignmentStateLocked).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
				AddRow(pcID, traderID, "IN_PROGRESS", templateID))

		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
			WithArgs(templateID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Test PC Template"))

		mockWM.On("GetWorkflowInstance", ctx, pcID).Return(&model.Workflow{
			BaseModel: model.BaseModel{ID: pcID},
			Status:    model.WorkflowStatusInProgress,
			WorkflowNodes: []model.WorkflowNode{
				{
					BaseModel:              model.BaseModel{ID: uuid.NewString()},
					WorkflowNodeTemplateID: nodeTemplateID,
					State:                  model.WorkflowNodeStateReady,
					WorkflowNodeTemplate: model.WorkflowNodeTemplate{
						BaseModel: model.BaseModel{ID: nodeTemplateID},
						Name:      "Node",
						Type:      "SIMPLE_FORM",
					},
				},
			},
		}, nil).Once()

		results, err := svc.GetPreConsignmentsByTraderID(ctx, traderID)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, pcID, results[0].ID)
		mockWM.AssertExpectations(t)
	})

	t.Run("Empty", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
			WithArgs(traderID, model.PreConsignmentStateLocked).
			WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}))

		results, err := svc.GetPreConsignmentsByTraderID(ctx, traderID)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestPreConsignmentService_GetTraderPreConsignments(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewPreConsignmentService(db, mockTP, mockWM)

	ctx := context.Background()
	traderID := "trader1"
	limit := 10
	offset := 0

	// Count Templates
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignment_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Find Templates
	templateID := uuid.NewString()
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" ORDER BY name ASC LIMIT \$1`).
		WithArgs(limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "Test Template"))

	// Find PreConsignments for Trader
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
			AddRow(uuid.NewString(), traderID, "IN_PROGRESS", templateID))

	result, err := svc.GetTraderPreConsignments(ctx, traderID, &offset, &limit)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Len(t, result.Items, 1)
}

func TestPreConsignmentService_GetTraderPreConsignments_CountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewPreConsignmentService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignment_templates"`).
		WillReturnError(errors.New("db error"))

	result, err := svc.GetTraderPreConsignments(ctx, traderID, nil, nil)
	assert.Error(t, err)
	assert.Equal(t, model.TraderPreConsignmentsResponseDTO{}, result)
}
