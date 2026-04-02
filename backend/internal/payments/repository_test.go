package payments

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn: db,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock
}

func TestRepository_Create(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaymentRepository(db)

	tx := &PaymentTransaction{
		ID:              uuid.New().String(),
		ReferenceNumber: "REF-123",
		TaskID:          "TASK-123",
		Amount:          decimal.NewFromFloat(100.0),
		Status:          PaymentStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "payment_transactions"`).
		WithArgs(tx.ID, tx.ReferenceNumber, tx.TaskID, tx.SessionID, tx.Amount, tx.Currency, tx.Status, tx.PaymentMethod, tx.ExpiryDate, sqlmock.AnyArg(), tx.CreatedAt, tx.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(context.Background(), tx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_GetByReferenceNumber(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaymentRepository(db)

	ref := "REF-123"

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "reference_number", "status"}).
			AddRow("uuid-1", ref, PaymentStatusPending)

		mock.ExpectQuery(`SELECT \* FROM "payment_transactions" WHERE reference_number = \$1`).
			WithArgs(ref, 1).
			WillReturnRows(rows)

		res, err := repo.GetByReferenceNumber(context.Background(), ref)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res == nil || res.ReferenceNumber != ref {
			t.Errorf("expected reference %s, got %v", ref, res)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM "payment_transactions" WHERE reference_number = \$1`).
			WithArgs("UNKNOWN", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		res, err := repo.GetByReferenceNumber(context.Background(), "UNKNOWN")
		if err != nil {
			t.Fatalf("expected no error (nil, nil) for not found, got %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result for not found, got %v", res)
		}
	})
}

func TestRepository_GetByTaskID(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaymentRepository(db)

	taskID := "TASK-123"

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "task_id", "status"}).
			AddRow("uuid-1", taskID, PaymentStatusPending)

		mock.ExpectQuery(`SELECT \* FROM "payment_transactions" WHERE task_id = \$1`).
			WithArgs(taskID, 1).
			WillReturnRows(rows)

		res, err := repo.GetByTaskID(context.Background(), taskID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res == nil || res.TaskID != taskID {
			t.Errorf("expected task id %s, got %v", taskID, res)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM "payment_transactions" WHERE task_id = \$1`).
			WithArgs("UNKNOWN", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		res, err := repo.GetByTaskID(context.Background(), "UNKNOWN")
		if err != nil {
			t.Fatalf("expected no error (nil, nil) for not found, got %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result for not found, got %v", res)
		}
	})
}

func TestRepository_Update(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaymentRepository(db)

	tx := &PaymentTransaction{
		ID:              "uuid-1",
		ReferenceNumber: "REF-123",
		Status:          PaymentStatusSuccess,
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "payment_transactions" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Update(context.Background(), tx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRepository_UpdateStatus(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewPaymentRepository(db)

	ref := "REF-123"
	status := PaymentStatusSuccess

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "payment_transactions" SET "status"=\$1,"updated_at"=\$2 WHERE reference_number = \$3`).
		WithArgs(status, sqlmock.AnyArg(), ref).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(context.Background(), ref, status)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRepository_WithTx(t *testing.T) {
	db, _ := setupTestDB(t)
	repo := NewPaymentRepository(db)

	newRepo := repo.WithTx(db)
	if newRepo == nil {
		t.Fatal("expected WithTx to return a new repository")
	}
}
