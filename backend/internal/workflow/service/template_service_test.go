package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a gorm database", err)
	}

	return gdb, mock
}

func TestTemplateService_GetWorkflowNodeTemplatesByIDs(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	ids := []string{id1, id2}

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE id IN \(\$1,\$2\)`).
		WithArgs(id1, id2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(id1, "Template 1").
			AddRow(id2, "Template 2"))

	result, err := service.GetWorkflowNodeTemplatesByIDs(ctx, ids)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTemplateService_GetWorkflowTemplateByIDV2(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewTemplateService(db)
	ctx := context.Background()

	id := uuid.NewString()

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

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_node_templates" WHERE id = \$1 ORDER BY "workflow_node_templates"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "Test Node Template"))

	result, err := service.GetWorkflowNodeTemplateByID(ctx, id)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, id, result.ID)
}
