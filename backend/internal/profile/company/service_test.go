package company

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var now = time.Now()

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	dialector := postgres.New(postgres.Config{Conn: db})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock
}

func setupPingTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mock.ExpectPing() // consumed by gorm.Open's connectivity check
	dialector := postgres.New(postgres.Config{Conn: db})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock
}

var companyColumns = []string{"id", "name", "ou_id", "ou_handle", "data", "created_at", "updated_at"}

func companyRow(id, name, ouId, ouHandle string, data []byte) *sqlmock.Rows {
	return sqlmock.NewRows(companyColumns).
		AddRow(id, name, ouId, ouHandle, data, now, now)
}

// --- GetCompanyByID ---

func TestService_GetCompanyByID_InvalidID(t *testing.T) {
	svc := NewService(nil)
	if _, err := svc.GetCompanyByID(context.Background(), ""); !errors.Is(err, ErrInvalidCompanyID) {
		t.Fatalf("expected ErrInvalidCompanyID, got %v", err)
	}
}

func TestService_GetCompanyByID_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE id = \$1`).
		WithArgs("missing-id", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	if _, err := svc.GetCompanyByID(context.Background(), "missing-id"); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_GetCompanyByID_DBError(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE id = \$1`).
		WithArgs("co-1", 1).
		WillReturnError(errors.New("query failed"))

	if _, err := svc.GetCompanyByID(context.Background(), "co-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_GetCompanyByID_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE id = \$1`).
		WithArgs("co-1", 1).
		WillReturnRows(companyRow("co-1", "Acme", "acme-id", "acme-handle", []byte(`{}`)))

	record, err := svc.GetCompanyByID(context.Background(), "co-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if record == nil || record.ID != "co-1" || record.Name != "Acme" {
		t.Fatalf("unexpected record: %#v", record)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- GetCompanyByOUId ---

func TestService_GetCompanyByOUId_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE ou_id = \$1`).
		WithArgs("missing-ouid", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	if _, err := svc.GetCompanyByOUId(context.Background(), "missing-ouid"); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_GetCompanyByOUId_DBError(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE ou_id = \$1`).
		WithArgs("acme-id", 1).
		WillReturnError(errors.New("query failed"))

	if _, err := svc.GetCompanyByOUId(context.Background(), "acme-id"); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_GetCompanyByOUId_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE ou_id = \$1`).
		WithArgs("acme-id", 1).
		WillReturnRows(companyRow("co-1", "Acme", "acme-id", "acme-handle", []byte(`{}`)))

	record, err := svc.GetCompanyByOUId(context.Background(), "acme-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if record == nil || record.OUID != "acme-id" || record.Name != "Acme" {
		t.Fatalf("unexpected record: %#v", record)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- GetCompanyByOUHandle ---

func TestService_GetCompanyByOUHandle_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE ou_handle = \$1`).
		WithArgs("missing-handle", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	if _, err := svc.GetCompanyByOUHandle(context.Background(), "missing-handle"); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_GetCompanyByOUHandle_DBError(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE ou_handle = \$1`).
		WithArgs("acme", 1).
		WillReturnError(errors.New("query failed"))

	if _, err := svc.GetCompanyByOUHandle(context.Background(), "acme"); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_GetCompanyByOUHandle_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectQuery(`SELECT .* FROM "company_records" WHERE ou_handle = \$1`).
		WithArgs("acme-handle", 1).
		WillReturnRows(companyRow("co-1", "Acme", "acme-id", "acme-handle", []byte(`{}`)))

	record, err := svc.GetCompanyByOUHandle(context.Background(), "acme-handle")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if record == nil || record.OUHandle != "acme-handle" {
		t.Fatalf("unexpected record: %#v", record)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- UpdateCompany ---

func TestService_UpdateCompany_InvalidID(t *testing.T) {
	svc := NewService(nil)
	if err := svc.UpdateCompany(context.Background(), "", map[string]any{"k": "v"}); !errors.Is(err, ErrInvalidCompanyID) {
		t.Fatalf("expected ErrInvalidCompanyID, got %v", err)
	}
}

func TestService_UpdateCompany_EmptyData(t *testing.T) {
	svc := NewService(nil)
	// Empty data is a no-op — no DB call should be made.
	if err := svc.UpdateCompany(context.Background(), "co-1", map[string]any{}); err != nil {
		t.Fatalf("expected no error for empty data, got %v", err)
	}
}

func TestService_UpdateCompany_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	// Atomic JSONB merge: no prior SELECT, UPDATE returns 0 rows when id is missing.
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "company_records" SET`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "missing-id").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := svc.UpdateCompany(context.Background(), "missing-id", map[string]any{"k": "v"}); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_UpdateCompany_DBError(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "company_records" SET`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "co-1").
		WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()

	if err := svc.UpdateCompany(context.Background(), "co-1", map[string]any{"new": "key"}); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_UpdateCompany_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "company_records" SET`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "co-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.UpdateCompany(context.Background(), "co-1", map[string]any{"new": "key"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- Health ---

func TestService_Health_Success(t *testing.T) {
	db, mock := setupPingTestDB(t)
	svc := NewService(db)

	mock.ExpectPing()

	if err := svc.Health(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestService_Health_DBError(t *testing.T) {
	db, mock := setupPingTestDB(t)
	svc := NewService(db)

	mock.ExpectPing().WillReturnError(errors.New("health failed"))

	if err := svc.Health(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
