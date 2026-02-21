package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func TestTemplateService_GetWorkflowTemplateByHSCodeIDAndFlow(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	hsCodeID := uuid.New()
	flow := model.ConsignmentFlowImport
	templateID := uuid.New()

	// Expectation
	sqlMock.ExpectQuery(`SELECT workflow_templates\.\* FROM "workflow_templates" JOIN workflow_template_maps ON workflow_templates\.id = workflow_template_maps\.workflow_template_id WHERE workflow_template_maps\.hs_code_id = \$1 AND workflow_template_maps\.consignment_flow = \$2 ORDER BY "workflow_templates"."id" LIMIT \$3`).
		WithArgs(hsCodeID, flow, 1). // Checking matches exact args
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "name"}).
			AddRow(templateID, flow, "Test Template"))

	result, err := service.GetWorkflowTemplateByHSCodeIDAndFlow(ctx, hsCodeID, flow)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, templateID, result.ID)
}

func TestTemplateService_GetWorkflowNodeTemplatesByIDs(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id1 := uuid.New()
	id2 := uuid.New()
	ids := []uuid.UUID{id1, id2}

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE id IN \(\$1,\$2\)`).
		WithArgs(id1, id2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(id1, "Template 1").
			AddRow(id2, "Template 2"))

	result, err := service.GetWorkflowNodeTemplatesByIDs(ctx, ids)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTemplateService_GetWorkflowTemplateByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id := uuid.New()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_templates" WHERE id = \$1 ORDER BY "workflow_templates"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "Test Template"))

	result, err := service.GetWorkflowTemplateByID(ctx, id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}

func TestTemplateService_GetWorkflowNodeTemplateByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id := uuid.New()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE id = \$1 ORDER BY "workflow_node_templates"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "Test Node Template"))

	result, err := service.GetWorkflowNodeTemplateByID(ctx, id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}
