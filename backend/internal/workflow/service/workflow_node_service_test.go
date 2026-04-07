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

func TestWorkflowNodeService_GetWorkflowNodeByIDInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()

	// We mock Begin() for the manual tx start below
	sqlMock.ExpectBegin()
	tx := db.Begin()

	nodeID := uuid.NewString()

	// Expectation
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
		WithArgs(nodeID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).
			AddRow(nodeID, "READY"))

	result, err := service.GetWorkflowNodeByIDInTx(ctx, tx, nodeID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, nodeID, result.ID)
}

func TestWorkflowNodeService_CreateWorkflowNodesInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	// Manually start tx (requires expectation)
	sqlMock.ExpectBegin()
	tx := db.Begin()

	nodes := []model.WorkflowNode{{BaseModel: model.BaseModel{ID: uuid.NewString()}}}

	// Expectation: Create
	sqlMock.ExpectExec(`INSERT INTO "workflow_nodes"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := service.CreateWorkflowNodesInTx(ctx, tx, nodes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestWorkflowNodeService_UpdateWorkflowNodesInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	nodeID := uuid.NewString()
	node := model.WorkflowNode{
		BaseModel: model.BaseModel{ID: nodeID},
		State:     model.WorkflowNodeStateCompleted,
	}
	nodes := []model.WorkflowNode{node}

	// Expectation: Select for update (First)
	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
		WithArgs(nodeID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "state"}).AddRow(nodeID, "IN_PROGRESS"))

	// Expectation: Save (Update)
	// Save updates all fields
	sqlMock.ExpectExec(`UPDATE "workflow_nodes" SET "created_at"=\$1,"updated_at"=\$2,"workflow_id"=\$3,"workflow_node_template_id"=\$4,"state"=\$5,"extended_state"=\$6,"outcome"=\$7,"depends_on"=\$8,"unlock_configuration"=\$9 WHERE "id" = \$10`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "COMPLETED", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nodeID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := service.UpdateWorkflowNodesInTx(ctx, tx, nodes)
	assert.NoError(t, err)
}

func TestWorkflowNodeService_GetWorkflowNodesByWorkflowIDInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	workflowID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE workflow_id = \$1`).
		WithArgs(workflowID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_id"}).AddRow(uuid.NewString(), workflowID))

	result, err := service.GetWorkflowNodesByWorkflowIDInTx(ctx, tx, workflowID)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestWorkflowNodeService_CountIncompleteNodesByWorkflowID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	workflowID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "workflow_nodes" WHERE workflow_id = \$1 AND state != \$2`).
		WithArgs(workflowID, model.WorkflowNodeStateCompleted).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	count, err := service.CountIncompleteNodesByWorkflowID(ctx, tx, workflowID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestWorkflowNodeService_GetWorkflowNodesByWorkflowIDInTx_PreConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	workflowID := uuid.NewString()
	nodeID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE workflow_id = \$1`).
		WithArgs(workflowID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_id"}).AddRow(nodeID, workflowID))

	nodes, err := service.GetWorkflowNodesByWorkflowIDInTx(ctx, tx, workflowID)
	assert.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, nodeID, nodes[0].ID)
}

func TestWorkflowNodeService_CountIncompleteNodesByWorkflowID_PreConsignment(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	workflowID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "workflow_nodes" WHERE workflow_id = \$1 AND state != \$2`).
		WithArgs(workflowID, model.WorkflowNodeStateCompleted).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	count, err := service.CountIncompleteNodesByWorkflowID(ctx, tx, workflowID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestWorkflowNodeService_GetWorkflowNodesByIDsInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	ids := []string{id1, id2}

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id IN \(\$1,\$2\)`).
		WithArgs(id1, id2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id1).AddRow(id2))

	nodes, err := service.GetWorkflowNodesByIDsInTx(ctx, tx, ids)
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestWorkflowNodeService_GetWorkflowNodesByWorkflowIDsInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	ids := []string{id1, id2}

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE workflow_id IN \(\$1,\$2\)`).
		WithArgs(id1, id2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_id"}).AddRow(uuid.NewString(), id1).AddRow(uuid.NewString(), id2))

	nodes, err := service.GetWorkflowNodesByWorkflowIDsInTx(ctx, tx, ids)
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestWorkflowNodeService_GetWorkflowNodeByIDInTx_Success(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()
	assert.NoError(t, tx.Error)
	defer func() {
		assert.NoError(t, tx.Rollback().Error)
	}()
	id := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
	sqlMock.ExpectRollback()

	node, err := service.GetWorkflowNodeByIDInTx(ctx, tx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, node.ID)
}

func TestWorkflowNodeService_UpdateWorkflowNodesInTx_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	nodeID := uuid.NewString()
	nodes := []model.WorkflowNode{{BaseModel: model.BaseModel{ID: nodeID}}}

	t.Run("Node Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
			WithArgs(nodeID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		err := service.UpdateWorkflowNodesInTx(ctx, tx, nodes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find workflow node")
	})
}

func TestWorkflowNodeService_GetWorkflowNodeByIDInTx_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	id := uuid.NewString()

	t.Run("Not Found", func(t *testing.T) {
		sqlMock.ExpectBegin()
		tx := db.Begin()
		assert.NoError(t, tx.Error)
		defer func() {
			assert.NoError(t, tx.Rollback().Error)
		}()

		sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
			WithArgs(id, 1).
			WillReturnError(gorm.ErrRecordNotFound)
		sqlMock.ExpectRollback()

		node, err := service.GetWorkflowNodeByIDInTx(ctx, tx, id)
		assert.Error(t, err)
		assert.Nil(t, node)
	})
}
