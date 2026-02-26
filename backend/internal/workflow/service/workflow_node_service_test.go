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

	nodeID := uuid.New()

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

	nodes := []model.WorkflowNode{{BaseModel: model.BaseModel{ID: uuid.New()}}}

	// Expectation: Create
	sqlMock.ExpectExec(`INSERT INTO "workflow_nodes"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
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

	nodeID := uuid.New()
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
	sqlMock.ExpectExec(`UPDATE "workflow_nodes" SET "created_at"=\$1,"updated_at"=\$2,"consignment_id"=\$3,"pre_consignment_id"=\$4,"workflow_node_template_id"=\$5,"state"=\$6,"extended_state"=\$7,"outcome"=\$8,"depends_on"=\$9,"unlock_configuration"=\$10 WHERE "id" = \$11`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "COMPLETED", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nodeID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := service.UpdateWorkflowNodesInTx(ctx, tx, nodes)
	assert.NoError(t, err)
}

func TestWorkflowNodeService_GetWorkflowNodesByConsignmentIDInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	consignmentID := uuid.New()

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE consignment_id = \$1`).
		WithArgs(consignmentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "consignment_id"}).AddRow(uuid.New(), consignmentID))

	result, err := service.GetWorkflowNodesByConsignmentIDInTx(ctx, tx, consignmentID)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestWorkflowNodeService_CountIncompleteNodesByConsignmentID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	consignmentID := uuid.New()

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "workflow_nodes" WHERE consignment_id = \$1 AND state != \$2`).
		WithArgs(consignmentID, model.WorkflowNodeStateCompleted).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	count, err := service.CountIncompleteNodesByConsignmentID(ctx, tx, consignmentID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestWorkflowNodeService_GetWorkflowNodesByPreConsignmentIDInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	pcID := uuid.New()
	nodeID := uuid.New()

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE pre_consignment_id = \$1`).
		WithArgs(pcID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "pre_consignment_id"}).AddRow(nodeID, pcID))

	nodes, err := service.GetWorkflowNodesByPreConsignmentIDInTx(ctx, tx, pcID)
	assert.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, nodeID, nodes[0].ID)
}

func TestWorkflowNodeService_CountIncompleteNodesByPreConsignmentID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	pcID := uuid.New()

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "workflow_nodes" WHERE pre_consignment_id = \$1 AND state != \$2`).
		WithArgs(pcID, model.WorkflowNodeStateCompleted).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	count, err := service.CountIncompleteNodesByPreConsignmentID(ctx, tx, pcID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestWorkflowNodeService_GetWorkflowNodesByIDsInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	id1 := uuid.New()
	id2 := uuid.New()
	ids := []uuid.UUID{id1, id2}

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id IN \(\$1,\$2\)`).
		WithArgs(id1, id2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id1).AddRow(id2))

	nodes, err := service.GetWorkflowNodesByIDsInTx(ctx, tx, ids)
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestWorkflowNodeService_GetWorkflowNodesByConsignmentIDsInTx(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	id1 := uuid.New()
	id2 := uuid.New()
	ids := []uuid.UUID{id1, id2}

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE consignment_id IN \(\$1,\$2\)`).
		WithArgs(id1, id2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "consignment_id"}).AddRow(uuid.New(), id1).AddRow(uuid.New(), id2))

	nodes, err := service.GetWorkflowNodesByConsignmentIDsInTx(ctx, tx, ids)
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestWorkflowNodeService_GetWorkflowNodeByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	id := uuid.New()

	sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

	node, err := service.GetWorkflowNodeByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, node.ID)
}

func TestWorkflowNodeService_UpdateWorkflowNodesInTx_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	sqlMock.ExpectBegin()
	tx := db.Begin()

	nodeID := uuid.New()
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

func TestWorkflowNodeService_GetWorkflowNodeByID_Failure(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	service := NewWorkflowNodeService(db)
	ctx := context.Background()
	id := uuid.New()

	t.Run("Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(`SELECT \* FROM "workflow_nodes" WHERE id = \$1 ORDER BY "workflow_nodes"."id" LIMIT \$2`).
			WithArgs(id, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		node, err := service.GetWorkflowNodeByID(ctx, id)
		assert.Error(t, err)
		assert.Nil(t, node)
	})
}
