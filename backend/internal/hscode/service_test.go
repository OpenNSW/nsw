package hscode

import (
	"context"
	"fmt"
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

func TestHSCodeService_GetAllHSCodes(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("Success - Default Pagination", func(t *testing.T) {
		filter := Filter{}

		// Find query first; 2 results < limit=50, so no separate count query needed
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" ORDER BY hs_code ASC LIMIT \$1`).
			WithArgs(50).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).
				AddRow(uuid.NewString(), "1234.56").
				AddRow(uuid.NewString(), "7890.12"))

		result, err := service.GetAll(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Items, 2)
	})

	t.Run("Success - With Filter", func(t *testing.T) {
		startsWith := "12"
		filter := Filter{
			HSCodeStartsWith: &startsWith,
		}

		// Find query first; 1 result < limit=50, so no separate count query needed
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE hs_code LIKE \$1 ORDER BY hs_code ASC LIMIT \$2`).
			WithArgs("12%", 50).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).
				AddRow(uuid.NewString(), "1234.56"))

		result, err := service.GetAll(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Len(t, result.Items, 1)
	})

	t.Run("Success - Empty Result", func(t *testing.T) {
		filter := Filter{}

		// Find query returns empty; 0 < limit=50, so no separate count query needed
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" ORDER BY hs_code ASC LIMIT \$1`).
			WithArgs(50).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}))

		result, err := service.GetAll(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Items)
	})
}

func TestHSCodeService_GetHSCodeByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()
	hsCodeID := uuid.NewString()

	t.Run("Success", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id = \$1 ORDER BY "hs_codes"."id" LIMIT \$2`).
			WithArgs(hsCodeID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

		result, err := service.GetByID(ctx, hsCodeID)
		assert.NoError(t, err)
		assert.Equal(t, hsCodeID, result.ID)
	})

	t.Run("Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id = \$1 ORDER BY "hs_codes"."id" LIMIT \$2`).
			WithArgs(hsCodeID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		result, err := service.GetByID(ctx, hsCodeID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Generic DB Error", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id = \$1 ORDER BY "hs_codes"."id" LIMIT \$2`).
			WithArgs(hsCodeID, 1).
			WillReturnError(fmt.Errorf("connection refused"))

		result, err := service.GetByID(ctx, hsCodeID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to retrieve HS code")
	})
}

func TestHSCodeService_GetAllHSCodes_FindError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" ORDER BY hs_code ASC LIMIT \$1`).
		WithArgs(50).
		WillReturnError(fmt.Errorf("connection lost"))

	result, err := service.GetAll(ctx, Filter{})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve HS codes")
}
