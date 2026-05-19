package preconsignment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a gorm database", err)
	}

	return gdb, mock
}

func TestPreConsignmentService_InitializePreConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewService(db, mockTP, mockWM)

	ctx := context.Background()
	traderID := "trader1"
	templateID := uuid.NewString()
	createReq := &CreatePreConsignmentDTO{
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
	svc := NewService(db, mockTP, mockWM)

	ctx := context.Background()
	templateID := uuid.NewString()
	createReq := &CreatePreConsignmentDTO{
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
	svc := NewService(db, mockTP, mockWM)

	ctx := context.Background()
	templateID := uuid.NewString()
	workflowTemplateID := uuid.NewString()
	createReq := &CreatePreConsignmentDTO{
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
	svc := NewService(db, nil, mockWM)

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
	svc := NewService(db, nil, mockWM)

	ctx := context.Background()
	traderID := "trader1"

	t.Run("Success", func(t *testing.T) {
		pcID := uuid.NewString()
		templateID := uuid.NewString()
		nodeTemplateID := uuid.NewString()

		sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
			WithArgs(traderID, StateLocked).
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
			WithArgs(traderID, StateLocked).
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
	svc := NewService(db, mockTP, mockWM)

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
	svc := NewService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignment_templates"`).
		WillReturnError(errors.New("db error"))

	result, err := svc.GetTraderPreConsignments(ctx, traderID, nil, nil)
	assert.Error(t, err)
	assert.Equal(t, TraderPreConsignmentsResponseDTO{}, result)
}

func TestPreConsignmentService_OnWorkflowStatusChanged_Completed(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	id := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "pre_consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err := svc.OnWorkflowStatusChanged(context.Background(), db, id,
		model.WorkflowStatusInProgress, model.WorkflowStatusCompleted,
		&model.Workflow{BaseModel: model.BaseModel{ID: id}, GlobalContext: map[string]any{}})
	assert.NoError(t, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPreConsignmentService_OnWorkflowStatusChanged_NonTerminal(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	id := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))

	err := svc.OnWorkflowStatusChanged(context.Background(), db, id,
		model.WorkflowStatusInProgress, model.WorkflowStatusFailed, nil)
	assert.NoError(t, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPreConsignmentService_OnWorkflowStatusChanged_NotFound(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	id := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	err := svc.OnWorkflowStatusChanged(context.Background(), db, id,
		model.WorkflowStatusInProgress, model.WorkflowStatusCompleted, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve pre-consignment")
}

func TestPreConsignmentService_OnWorkflowStatusChanged_CompletedNilWorkflow(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	id := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(id, "IN_PROGRESS"))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`UPDATE "pre_consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err := svc.OnWorkflowStatusChanged(context.Background(), db, id,
		model.WorkflowStatusInProgress, model.WorkflowStatusCompleted, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow payload cannot be nil")
}

func TestPreConsignmentService_InitializePreConsignment_NilRequest(t *testing.T) {
	svc := NewService(nil, nil, nil)
	_, err := svc.InitializePreConsignment(context.Background(), nil, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create request cannot be nil")
}

func TestPreConsignmentService_InitializePreConsignment_EmptyTraderID(t *testing.T) {
	svc := NewService(nil, nil, nil)
	_, err := svc.InitializePreConsignment(context.Background(), &CreatePreConsignmentDTO{}, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trader ID cannot be empty")
}

func TestPreConsignmentService_InitializePreConsignment_DependencyNotCompleted(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	ctx := context.Background()
	templateID := uuid.NewString()
	depID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1`).
		WithArgs(templateID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
			AddRow(templateID, uuid.NewString(), []byte(`["`+depID+`"]`)))

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignments"`).
		WithArgs("trader1", depID, StateCompleted).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	_, err := svc.InitializePreConsignment(ctx, &CreatePreConsignmentDTO{PreConsignmentTemplateID: templateID}, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency pre-consignments are not all completed")
}

func TestPreConsignmentService_InitializePreConsignment_DependencyCheckError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	ctx := context.Background()
	templateID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1`).
		WithArgs(templateID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
			AddRow(templateID, uuid.NewString(), []byte(`["`+uuid.NewString()+`"]`)))

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignments"`).
		WillReturnError(errors.New("db down"))

	_, err := svc.InitializePreConsignment(ctx, &CreatePreConsignmentDTO{PreConsignmentTemplateID: templateID}, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check dependency completion")
}

func TestPreConsignmentService_InitializePreConsignment_StartWorkflowError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockTP := new(MockTemplateProvider)
	mockWM := new(MockWorkflowManager)
	svc := NewService(db, mockTP, mockWM)
	ctx := context.Background()
	templateID := uuid.NewString()
	workflowTemplateID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE id = \$1`).
		WithArgs(templateID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_template_id", "depends_on"}).
			AddRow(templateID, workflowTemplateID, []byte("[]")))

	mockTP.On("GetWorkflowTemplateByID", ctx, workflowTemplateID).
		Return(&model.WorkflowTemplate{BaseModel: model.BaseModel{ID: workflowTemplateID}}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "pre_consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mockWM.On("StartWorkflowInstance", ctx, mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("wm failed"))
	sqlMock.ExpectRollback()

	_, err := svc.InitializePreConsignment(ctx, &CreatePreConsignmentDTO{PreConsignmentTemplateID: templateID}, "trader1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register workflow")
}

func TestPreConsignmentService_GetPreConsignmentsByTraderID_WorkflowError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	svc := NewService(db, nil, mockWM)
	ctx := context.Background()
	traderID := "trader1"
	pcID := uuid.NewString()
	templateID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
		WithArgs(traderID, StateLocked).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
			AddRow(pcID, traderID, "IN_PROGRESS", templateID))
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "T"))

	mockWM.On("GetWorkflowInstance", ctx, pcID).Return(nil, errors.New("wm down"))

	_, err := svc.GetPreConsignmentsByTraderID(ctx, traderID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow details")
}

func TestPreConsignmentService_GetPreConsignmentsByTraderID_QueryError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	ctx := context.Background()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1 AND state != \$2`).
		WillReturnError(errors.New("db down"))

	_, err := svc.GetPreConsignmentsByTraderID(ctx, "trader1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve pre-consignments")
}

func TestPreConsignmentService_GetPreConsignmentByID_WorkflowError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWorkflowManager)
	svc := NewService(db, nil, mockWM)
	ctx := context.Background()
	pcID := uuid.NewString()
	templateID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE id = \$1`).
		WithArgs(pcID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_template_id"}).AddRow(pcID, templateID))
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" WHERE "pre_consignment_templates"."id" = \$1`).
		WithArgs(templateID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(templateID, "T"))
	mockWM.On("GetWorkflowInstance", ctx, pcID).Return(nil, errors.New("wm down"))

	_, err := svc.GetPreConsignmentByID(ctx, pcID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get workflow details")
}

func TestPreConsignmentService_GetTraderPreConsignments_LockedAndCompletedDependencies(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"
	limit := 10
	offset := 0
	parentID := uuid.NewString()
	lockedID := uuid.NewString()
	readyID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "pre_consignment_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignment_templates" ORDER BY name ASC LIMIT \$1`).
		WithArgs(limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "depends_on"}).
			AddRow(parentID, "Parent", []byte("[]")).
			AddRow(lockedID, "Locked Child", []byte(`["`+parentID+`"]`)). // depends on incomplete parent
			AddRow(readyID, "Ready", []byte("[]")))

	// trader has the parent in IN_PROGRESS (not COMPLETED → child stays LOCKED)
	sqlMock.ExpectQuery(`SELECT \* FROM "pre_consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "trader_id", "state", "pre_consignment_template_id"}).
			AddRow(uuid.NewString(), traderID, "IN_PROGRESS", parentID))

	result, err := svc.GetTraderPreConsignments(ctx, traderID, &offset, &limit)
	assert.NoError(t, err)
	assert.Len(t, result.Items, 3)

	states := map[string]State{}
	for _, item := range result.Items {
		states[item.ID] = item.State
	}
	assert.Equal(t, State("IN_PROGRESS"), states[parentID]) // existing record's state surfaces
	assert.Equal(t, StateLocked, states[lockedID])
	assert.Equal(t, StateReady, states[readyID])
}
