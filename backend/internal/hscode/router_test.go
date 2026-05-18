package hscode

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHSCodeRouter_HandleGetAllHSCodes(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db)
	r := NewRouter(svc)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("(?i)SELECT .* FROM \"hs_codes\"").WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(uuid.NewString(), "1234.56"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := httptest.NewRecorder()
	r.HandleGetAll(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes_ServiceError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewService(db)
	r := NewRouter(svc)

	sqlMock.ExpectQuery("(?i)SELECT count").WillReturnError(fmt.Errorf("db error"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes", nil)
	w := httptest.NewRecorder()
	r.HandleGetAll(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes_PaginationError(t *testing.T) {
	db, _ := setupTestDB(t)
	r := NewRouter(NewService(db))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes?limit=invalid", nil)
	w := httptest.NewRecorder()
	r.HandleGetAll(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes_AllQueryParams(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	r := NewRouter(NewService(db))

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "hs_codes" WHERE hs_code LIKE \$1`).
		WithArgs("12%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE hs_code LIKE \$1 ORDER BY hs_code ASC LIMIT \$2 OFFSET \$3`).
		WithArgs("12%", 25, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code"}).AddRow(uuid.NewString(), "1234.56"))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes?hsCodeStartsWith=12&limit=25&offset=5", nil)
	w := httptest.NewRecorder()
	r.HandleGetAll(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHSCodeRouter_HandleGetAllHSCodes_InvalidOffset(t *testing.T) {
	db, _ := setupTestDB(t)
	r := NewRouter(NewService(db))

	req, _ := http.NewRequest("GET", "/api/v1/hscodes?offset=notanint", nil)
	w := httptest.NewRecorder()
	r.HandleGetAll(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
