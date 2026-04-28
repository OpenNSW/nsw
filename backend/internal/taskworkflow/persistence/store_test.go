package persistence

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/task/plugin"
)

func setupStoreTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestNewTaskWorkflowStore(t *testing.T) {
	t.Run("rejects nil db", func(t *testing.T) {
		store, err := NewTaskWorkflowStore(nil)

		assert.Nil(t, store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection cannot be nil")
	})

	t.Run("returns store", func(t *testing.T) {
		db, mock := setupStoreTestDB(t)

		store, err := NewTaskWorkflowStore(db)

		assert.NoError(t, err)
		assert.NotNil(t, store)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTaskWorkflowStoreCreate(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	task := &TaskWorkflowTask{
		TaskID:          "task-1",
		MacroWorkflowID: "macro-1",
		TaskTemplateID:  "template-1",
		State:           plugin.Initialized,
		Data:            json.RawMessage(`{"screen":"form"}`),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "task_workflow_tasks"`).
		WithArgs(
			task.TaskID,
			task.MacroWorkflowID,
			task.TaskTemplateID,
			task.State,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = store.Create(task)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreGetByTaskID(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	data := []byte(`{"screen":"form"}`)
	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Minute)

	mock.ExpectQuery(`SELECT \* FROM "task_workflow_tasks" WHERE task_id = \$1 ORDER BY "task_workflow_tasks"."task_id" LIMIT \$2`).
		WithArgs("task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"task_id",
			"macro_workflow_id",
			"task_template_id",
			"state",
			"data",
			"created_at",
			"updated_at",
		}).AddRow(
			"task-1",
			"macro-1",
			"template-1",
			plugin.InProgress,
			data,
			createdAt,
			updatedAt,
		))

	task, err := store.GetByTaskID("task-1")

	assert.NoError(t, err)
	assert.Equal(t, "task-1", task.TaskID)
	assert.Equal(t, "macro-1", task.MacroWorkflowID)
	assert.Equal(t, "template-1", task.TaskTemplateID)
	assert.Equal(t, plugin.InProgress, task.State)
	assert.JSONEq(t, string(data), string(task.Data))
	assert.Equal(t, createdAt, task.CreatedAt)
	assert.Equal(t, updatedAt, task.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreGetStateByTaskID(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT "state" FROM "task_workflow_tasks" WHERE task_id = \$1 ORDER BY "task_workflow_tasks"."task_id" LIMIT \$2`).
		WithArgs("task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"state"}).AddRow(plugin.Completed))

	state, err := store.GetStateByTaskID("task-1")

	assert.NoError(t, err)
	assert.Equal(t, plugin.Completed, state)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreGetDataByTaskID(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	data := []byte(`{"field":"value"}`)

	mock.ExpectQuery(`SELECT "data" FROM "task_workflow_tasks" WHERE task_id = \$1 ORDER BY "task_workflow_tasks"."task_id" LIMIT \$2`).
		WithArgs("task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"data"}).AddRow(data))

	got, err := store.GetDataByTaskID("task-1")

	assert.NoError(t, err)
	assert.JSONEq(t, string(data), string(got))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreGetByMacroWorkflowID(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT \* FROM "task_workflow_tasks" WHERE macro_workflow_id = \$1`).
		WithArgs("macro-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"task_id",
			"macro_workflow_id",
			"task_template_id",
			"state",
			"data",
		}).
			AddRow("task-1", "macro-1", "template-1", plugin.InProgress, []byte(`{"step":1}`)).
			AddRow("task-2", "macro-1", "template-2", plugin.Completed, []byte(`{"step":2}`)))

	tasks, err := store.GetByMacroWorkflowID("macro-1")

	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "task-1", tasks[0].TaskID)
	assert.Equal(t, "task-2", tasks[1].TaskID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreUpdate(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	task := &TaskWorkflowTask{
		TaskID:          "task-1",
		MacroWorkflowID: "macro-1",
		TaskTemplateID:  "template-1",
		State:           plugin.Completed,
		Data:            json.RawMessage(`{"done":true}`),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "task_workflow_tasks" SET`).
		WithArgs(
			task.TaskID,
			task.MacroWorkflowID,
			task.TaskTemplateID,
			task.State,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			task.TaskID,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = store.Update(task)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreUpdateReturnsNotFoundWhenNoRowsUpdated(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	task := &TaskWorkflowTask{
		TaskID:          "missing-task",
		MacroWorkflowID: "macro-1",
		TaskTemplateID:  "template-1",
		State:           plugin.Completed,
		Data:            json.RawMessage(`{"done":true}`),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "task_workflow_tasks" SET`).
		WithArgs(
			task.TaskID,
			task.MacroWorkflowID,
			task.TaskTemplateID,
			task.State,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			task.TaskID,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err = store.Update(task)

	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreUpdateState(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "task_workflow_tasks" SET "state"=\$1,"updated_at"=\$2 WHERE task_id = \$3`).
		WithArgs(plugin.Failed, sqlmock.AnyArg(), "task-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = store.UpdateState("task-1", plugin.Failed)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreUpdateData(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	data := json.RawMessage(`{"field":"updated"}`)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "task_workflow_tasks" SET "data"=\$1,"updated_at"=\$2 WHERE task_id = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "task-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = store.UpdateData("task-1", data)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreDelete(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "task_workflow_tasks" WHERE task_id = \$1`).
		WithArgs("task-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = store.Delete("task-1")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskWorkflowStoreDeleteReturnsNotFoundWhenNoRowsDeleted(t *testing.T) {
	db, mock := setupStoreTestDB(t)
	store, err := NewTaskWorkflowStore(db)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "task_workflow_tasks" WHERE task_id = \$1`).
		WithArgs("missing-task").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err = store.Delete("missing-task")

	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}
