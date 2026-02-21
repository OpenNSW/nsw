package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func TestHSCodeService_GetAllHSCodes(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewHSCodeService(db)
	ctx := context.Background()

	t.Run("Success - Default Pagination", func(t *testing.T) {
		filter := model.HSCodeFilter{}

		// Count query
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "hs_codes"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		// Find query
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" ORDER BY hs_code ASC LIMIT \$1`).
			WithArgs(50). // Default limit
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).
				AddRow(uuid.New(), "1234.56").
				AddRow(uuid.New(), "7890.12"))

		result, err := service.GetAllHSCodes(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), result.TotalCount)
		assert.Len(t, result.Items, 2)
	})

	t.Run("Success - With Filter", func(t *testing.T) {
		startsWith := "12"
		filter := model.HSCodeFilter{
			HSCodeStartsWith: &startsWith,
		}

		// Count query with filter
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "hs_codes" WHERE hs_code LIKE \$1`).
			WithArgs("12%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Find query with filter
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE hs_code LIKE \$1 ORDER BY hs_code ASC LIMIT \$2`).
			WithArgs("12%", 50).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).
				AddRow(uuid.New(), "1234.56"))

		result, err := service.GetAllHSCodes(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalCount)
		assert.Len(t, result.Items, 1)
	})

	t.Run("Success - Empty Result", func(t *testing.T) {
		filter := model.HSCodeFilter{}

		// Count query returns 0
		sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "hs_codes"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Find query should NOT be executed

		result, err := service.GetAllHSCodes(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), result.TotalCount)
		assert.Empty(t, result.Items)
	})
}

func TestHSCodeService_GetHSCodeByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewHSCodeService(db)
	ctx := context.Background()
	hsCodeID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id = \$1 ORDER BY "hs_codes"."id" LIMIT \$2`).
			WithArgs(hsCodeID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(hsCodeID, "1234.56"))

		result, err := service.GetHSCodeByID(ctx, hsCodeID)
		assert.NoError(t, err)
		assert.Equal(t, hsCodeID, result.ID)
	})

	t.Run("Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id = \$1 ORDER BY "hs_codes"."id" LIMIT \$2`).
			WithArgs(hsCodeID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		result, err := service.GetHSCodeByID(ctx, hsCodeID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}
