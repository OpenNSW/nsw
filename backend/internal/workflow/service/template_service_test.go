package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTemplateService_GetWorkflowNodeTemplatesByIDs(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	ids := []string{id1, id2}

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

	id := uuid.NewString()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_templates" WHERE id = \$1 ORDER BY "workflow_templates"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "Test Template"))

	result, err := service.GetWorkflowTemplateByID(ctx, id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}

func TestTemplateService_GetWorkflowTemplateByIDV2(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id := uuid.NewString()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_template_v2" WHERE id = \$1 ORDER BY "workflow_template_v2"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "Test Template V2"))

	result, err := service.GetWorkflowTemplateByIDV2(ctx, id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}

func TestTemplateService_GetWorkflowNodeTemplateByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id := uuid.NewString()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE id = \$1 ORDER BY "workflow_node_templates"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "Test Node Template"))

	result, err := service.GetWorkflowNodeTemplateByID(ctx, id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}
