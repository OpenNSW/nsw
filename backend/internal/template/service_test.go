package template

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

func TestTemplateService_GetWorkflowTemplateByHSCodeIDAndFlow(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewTemplateService(db)
	ctx := context.Background()

	hsCodeID := uuid.NewString()
	flow := "IMPORT"
	templateID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT workflow_templates\.\* FROM "workflow_templates" JOIN workflow_template_maps ON workflow_templates\.id = workflow_template_maps\.workflow_template_id WHERE workflow_template_maps\.hs_code_id = \$1 AND workflow_template_maps\.consignment_flow = \$2 ORDER BY "workflow_templates"."id" LIMIT \$3`).
		WithArgs(hsCodeID, flow, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "name"}).
			AddRow(templateID, flow, "Test Template"))

	result, err := svc.GetWorkflowTemplateByHSCodeIDAndFlow(ctx, hsCodeID, flow)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, templateID, result.ID)
}

func TestTemplateService_GetWorkflowTemplateByHSCodeIDAndFlow_InvalidFlow(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewTemplateService(db)
	ctx := context.Background()

	result, err := svc.GetWorkflowTemplateByHSCodeIDAndFlow(ctx, uuid.NewString(), "INVALID")
	assert.Error(t, err)
	assert.Nil(t, result)
}
