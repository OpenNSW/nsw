package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

func strPtr(s string) *string { return &s }

// MockWorkflowNodeRepository is a mock implementation of WorkflowNodeRepository
type MockWorkflowNodeRepository struct {
	mock.Mock
}

func (m *MockWorkflowNodeRepository) GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*model.WorkflowNode, error) {
	args := m.Called(ctx, tx, nodeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNode), args.Error(1)
}

func (m *MockWorkflowNodeRepository) GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, nodeIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}
func (m *MockWorkflowNodeRepository) CreateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, nodes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}

func (m *MockWorkflowNodeRepository) UpdateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) error {
	args := m.Called(ctx, tx, nodes)
	return args.Error(0)
}

func (m *MockWorkflowNodeRepository) GetWorkflowNodesByConsignmentIDInTx(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, consignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}

func (m *MockWorkflowNodeRepository) GetWorkflowNodesByConsignmentIDsInTx(ctx context.Context, tx *gorm.DB, consignmentIDs []uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, consignmentIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}

func (m *MockWorkflowNodeRepository) CountIncompleteNodesByConsignmentID(ctx context.Context, tx *gorm.DB, consignmentID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tx, consignmentID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWorkflowNodeRepository) GetWorkflowNodesByPreConsignmentIDInTx(ctx context.Context, tx *gorm.DB, preConsignmentID uuid.UUID) ([]model.WorkflowNode, error) {
	args := m.Called(ctx, tx, preConsignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNode), args.Error(1)
}

func (m *MockWorkflowNodeRepository) CountIncompleteNodesByPreConsignmentID(ctx context.Context, tx *gorm.DB, preConsignmentID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tx, preConsignmentID)
	return args.Get(0).(int64), args.Error(1)
}

func TestTransitionToCompleted(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Already Completed", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateCompleted,
		}
		result, err := sm.TransitionToCompleted(ctx, nil, node, nil)
		assert.NoError(t, err)
		assert.Empty(t, result.UpdatedNodes)
		assert.Empty(t, result.NewReadyNodes)
		assert.False(t, result.AllNodesCompleted)
	})

	t.Run("Invalid State Transition", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateLocked,
		}
		_, err := sm.TransitionToCompleted(ctx, nil, node, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transition")
	})

	t.Run("Successful Transition No Dependencies", func(t *testing.T) {
		nodeID := uuid.New()
		consignmentID := uuid.New()
		node := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}
		extendedState := "{\"foo\": \"bar\"}"
		updateReq := &model.UpdateWorkflowNodeDTO{
			ExtendedState: &extendedState,
		}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*node}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.AnythingOfType("[]model.WorkflowNode")).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, node, updateReq)
		assert.NoError(t, err)
		assert.Len(t, result.UpdatedNodes, 1)
		assert.Equal(t, model.WorkflowNodeStateCompleted, result.UpdatedNodes[0].State)
		assert.True(t, result.AllNodesCompleted)
	})

	t.Run("Unlock Dependent Nodes", func(t *testing.T) {
		nodeID := uuid.New()
		dependentNodeID := uuid.New()
		consignmentID := uuid.New()

		node := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		dependentNode := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: dependentNodeID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateLocked,
			DependsOn:     model.UUIDArray{nodeID},
		}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*node, *dependentNode}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 2
		})).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, node, &model.UpdateWorkflowNodeDTO{})
		assert.NoError(t, err)
		assert.Len(t, result.UpdatedNodes, 2)
		assert.Len(t, result.NewReadyNodes, 1)
		assert.Equal(t, dependentNodeID, result.NewReadyNodes[0].ID)
		assert.False(t, result.AllNodesCompleted)
	})
}

func TestInitializeNodesFromTemplates(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Create Nodes With Dependencies", func(t *testing.T) {
		template1ID := uuid.New()
		template2ID := uuid.New()

		templates := []model.WorkflowNodeTemplate{
			{
				BaseModel: model.BaseModel{ID: template1ID},
			},
			{
				BaseModel: model.BaseModel{ID: template2ID},
				DependsOn: model.UUIDArray{template1ID},
			},
		}

		parentRef := ParentRef{
			ConsignmentID: &uuid.UUID{},
		}

		// Mock CreateWorkflowNodesInTx
		mockRepo.On("CreateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 2
		})).Return([]model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.New()},
				WorkflowNodeTemplateID: template1ID,
				State:                  model.WorkflowNodeStateLocked,
			},
			{
				BaseModel:              model.BaseModel{ID: uuid.New()},
				WorkflowNodeTemplateID: template2ID,
				State:                  model.WorkflowNodeStateLocked,
			},
		}, nil).Once()

		// Mock UpdateWorkflowNodesInTx
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			// One node should be updated to READY (the one with no dependencies)
			// The other node should be updated with dependencies but remain LOCKED
			return len(nodes) == 2
		})).Return(nil).Once()

		createdNodes, newReadyNodes, err := sm.InitializeNodesFromTemplates(ctx, nil, parentRef, templates)
		assert.NoError(t, err)
		assert.Len(t, createdNodes, 2)
		assert.Len(t, newReadyNodes, 1)

		// Verify the node without dependencies is READY
		assert.Len(t, newReadyNodes, 1)
		assert.Equal(t, template1ID, newReadyNodes[0].WorkflowNodeTemplateID)

		var foundInCreated bool
		for _, node := range createdNodes {
			if node.WorkflowNodeTemplateID == template1ID {
				assert.Equal(t, model.WorkflowNodeStateReady, node.State)
				foundInCreated = true
			}
		}
		assert.True(t, foundInCreated, "ready node not found in createdNodes with correct state")
	})

	t.Run("Create Nodes Without Dependencies", func(t *testing.T) {
		templateID := uuid.New()
		templates := []model.WorkflowNodeTemplate{
			{BaseModel: model.BaseModel{ID: templateID}},
		}
		parentRef := ParentRef{ConsignmentID: &uuid.UUID{}}

		mockRepo.On("CreateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 1
		})).Return([]model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: uuid.New()},
				WorkflowNodeTemplateID: templateID,
				State:                  model.WorkflowNodeStateLocked,
			},
		}, nil).Once()

		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.Anything).Return(nil).Once()

		createdNodes, newReadyNodes, err := sm.InitializeNodesFromTemplates(ctx, nil, parentRef, templates)
		assert.NoError(t, err)
		assert.Len(t, createdNodes, 1)
		assert.Len(t, newReadyNodes, 1)
		assert.Equal(t, model.WorkflowNodeStateReady, createdNodes[0].State)
	})
}

func TestTransitionToFailed(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateInProgress,
		}
		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 1 && nodes[0].ID == node.ID && nodes[0].State == model.WorkflowNodeStateFailed
		})).Return(nil).Once()

		err := sm.TransitionToFailed(ctx, nil, node, updateReq)
		assert.NoError(t, err)
		assert.Equal(t, model.WorkflowNodeStateFailed, node.State)
	})

	t.Run("Already Failed", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateFailed,
		}

		err := sm.TransitionToFailed(ctx, nil, node, nil)
		assert.NoError(t, err)
		assert.Equal(t, model.WorkflowNodeStateFailed, node.State)
	})

	t.Run("Invalid Transition", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateLocked,
		}

		err := sm.TransitionToFailed(ctx, nil, node, nil)
		assert.Error(t, err)
	})
}

func TestTransitionToInProgress(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateReady,
		}
		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 1 && nodes[0].ID == node.ID && nodes[0].State == model.WorkflowNodeStateInProgress
		})).Return(nil).Once()

		err := sm.TransitionToInProgress(ctx, nil, node, updateReq)
		assert.NoError(t, err)
		assert.Equal(t, model.WorkflowNodeStateInProgress, node.State)
	})

	t.Run("Already InProgress", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateInProgress,
		}

		err := sm.TransitionToInProgress(ctx, nil, node, &model.UpdateWorkflowNodeDTO{})
		assert.NoError(t, err)
		assert.Equal(t, model.WorkflowNodeStateInProgress, node.State)
	})

	t.Run("Invalid Transition", func(t *testing.T) {
		node := &model.WorkflowNode{
			BaseModel: model.BaseModel{ID: uuid.New()},
			State:     model.WorkflowNodeStateLocked,
		}

		err := sm.TransitionToInProgress(ctx, nil, node, &model.UpdateWorkflowNodeDTO{})
		assert.Error(t, err)
	})
}

func TestTransitionToCompletedWithOutcome(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Outcome Set On Completion", func(t *testing.T) {
		nodeID := uuid.New()
		consignmentID := uuid.New()
		node := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}
		outcome := "APPROVED"
		updateReq := &model.UpdateWorkflowNodeDTO{
			Outcome: &outcome,
		}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*node}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 1 && nodes[0].Outcome != nil && *nodes[0].Outcome == "APPROVED"
		})).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, node, updateReq)
		assert.NoError(t, err)
		assert.Len(t, result.UpdatedNodes, 1)
		assert.Equal(t, model.WorkflowNodeStateCompleted, result.UpdatedNodes[0].State)
		assert.NotNil(t, result.UpdatedNodes[0].Outcome)
		assert.Equal(t, "APPROVED", *result.UpdatedNodes[0].Outcome)
	})
}

func TestUnlockWithUnlockConfiguration(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Condition Met - Unlock Dependent", func(t *testing.T) {
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		nodeA := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeAID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		// Node B has an UnlockConfiguration that requires Node A to be COMPLETED with outcome APPROVED
		nodeB := model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeBID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateLocked,
			DependsOn:     model.UUIDArray{nodeAID},
			UnlockConfiguration: &model.UnlockConfig{
				AnyOf: []model.UnlockGroup{
					{
						AllOf: []model.UnlockCondition{
							{NodeTemplateID: nodeAID, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
						},
					},
				},
			},
		}

		outcome := "APPROVED"
		updateReq := &model.UpdateWorkflowNodeDTO{
			Outcome: &outcome,
		}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 2
		})).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq)
		assert.NoError(t, err)
		assert.Len(t, result.NewReadyNodes, 1)
		assert.Equal(t, nodeBID, result.NewReadyNodes[0].ID)
	})

	t.Run("Condition Not Met - Wrong Outcome", func(t *testing.T) {
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		nodeA := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeAID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		nodeB := model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeBID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateLocked,
			DependsOn:     model.UUIDArray{nodeAID},
			UnlockConfiguration: &model.UnlockConfig{
				AnyOf: []model.UnlockGroup{
					{
						AllOf: []model.UnlockCondition{
							{NodeTemplateID: nodeAID, Outcome: strPtr("APPROVED")},
						},
					},
				},
			},
		}

		outcome := "REJECTED"
		updateReq := &model.UpdateWorkflowNodeDTO{
			Outcome: &outcome,
		}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			// Only node A should be updated (to COMPLETED), node B stays LOCKED
			return len(nodes) == 1
		})).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq)
		assert.NoError(t, err)
		assert.Empty(t, result.NewReadyNodes, "node B should not be unlocked with wrong outcome")
	})

	t.Run("OR Condition - Second Group Met", func(t *testing.T) {
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		nodeA := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeAID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		// Node B unlocks if A has outcome APPROVED OR A has outcome FAST_TRACKED
		nodeB := model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeBID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateLocked,
			DependsOn:     model.UUIDArray{nodeAID},
			UnlockConfiguration: &model.UnlockConfig{
				AnyOf: []model.UnlockGroup{
					{
						AllOf: []model.UnlockCondition{
							{NodeTemplateID: nodeAID, Outcome: strPtr("APPROVED")},
						},
					},
					{
						AllOf: []model.UnlockCondition{
							{NodeTemplateID: nodeAID, Outcome: strPtr("FAST_TRACKED")},
						},
					},
				},
			},
		}

		outcome := "FAST_TRACKED"
		updateReq := &model.UpdateWorkflowNodeDTO{
			Outcome: &outcome,
		}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 2
		})).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq)
		assert.NoError(t, err)
		assert.Len(t, result.NewReadyNodes, 1)
		assert.Equal(t, nodeBID, result.NewReadyNodes[0].ID)
	})

	t.Run("State Only Condition - No Outcome Required", func(t *testing.T) {
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		nodeA := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeAID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		// Node B only requires Node A to be COMPLETED (no outcome check)
		nodeB := model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeBID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateLocked,
			DependsOn:     model.UUIDArray{nodeAID},
			UnlockConfiguration: &model.UnlockConfig{
				AnyOf: []model.UnlockGroup{
					{
						AllOf: []model.UnlockCondition{
							{NodeTemplateID: nodeAID, State: strPtr("COMPLETED")},
						},
					},
				},
			},
		}

		// Complete without any outcome
		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 2
		})).Return(nil).Once()

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq)
		assert.NoError(t, err)
		assert.Len(t, result.NewReadyNodes, 1, "node B should unlock when A is COMPLETED regardless of outcome")
	})
}

func TestEndNodeWorkflowCompletion(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("End Node Completed - Workflow Done", func(t *testing.T) {
		endNodeTemplateID := uuid.New()
		otherTemplateID := uuid.New()
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		// Node A is the end node and is being completed
		nodeA := &model.WorkflowNode{
			BaseModel:              model.BaseModel{ID: nodeAID},
			ConsignmentID:          &consignmentID,
			WorkflowNodeTemplateID: endNodeTemplateID,
			State:                  model.WorkflowNodeStateInProgress,
		}

		// Node B is NOT the end node and is still locked
		nodeB := model.WorkflowNode{
			BaseModel:              model.BaseModel{ID: nodeBID},
			ConsignmentID:          &consignmentID,
			WorkflowNodeTemplateID: otherTemplateID,
			State:                  model.WorkflowNodeStateLocked,
			DependsOn:              model.UUIDArray{},
		}

		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.AnythingOfType("[]model.WorkflowNode")).Return(nil).Once()

		completionConfig := &WorkflowCompletionConfig{
			EndNodeTemplateID: &endNodeTemplateID,
		}

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq, completionConfig)
		assert.NoError(t, err)
		assert.True(t, result.AllNodesCompleted, "workflow should be complete when end node is completed")
	})

	t.Run("Non-End Node Completed - Workflow Not Done", func(t *testing.T) {
		endNodeTemplateID := uuid.New()
		otherTemplateID := uuid.New()
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		// Node A is NOT the end node, is being completed
		nodeA := &model.WorkflowNode{
			BaseModel:              model.BaseModel{ID: nodeAID},
			ConsignmentID:          &consignmentID,
			WorkflowNodeTemplateID: otherTemplateID,
			State:                  model.WorkflowNodeStateInProgress,
		}

		// Node B IS the end node, still locked
		nodeB := model.WorkflowNode{
			BaseModel:              model.BaseModel{ID: nodeBID},
			ConsignmentID:          &consignmentID,
			WorkflowNodeTemplateID: endNodeTemplateID,
			State:                  model.WorkflowNodeStateLocked,
			DependsOn:              model.UUIDArray{nodeAID},
		}

		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.AnythingOfType("[]model.WorkflowNode")).Return(nil).Once()

		completionConfig := &WorkflowCompletionConfig{
			EndNodeTemplateID: &endNodeTemplateID,
		}

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq, completionConfig)
		assert.NoError(t, err)
		assert.False(t, result.AllNodesCompleted, "workflow should not be complete when end node is still locked")
	})

	t.Run("No EndNodeTemplateID - Falls Back To All Nodes", func(t *testing.T) {
		nodeAID := uuid.New()
		nodeBID := uuid.New()
		consignmentID := uuid.New()

		nodeA := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeAID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		nodeB := model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeBID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateLocked,
			DependsOn:     model.UUIDArray{nodeAID},
		}

		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA, nodeB}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.AnythingOfType("[]model.WorkflowNode")).Return(nil).Once()

		// No completion config (nil) — should fall back to all-nodes-completed check
		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq)
		assert.NoError(t, err)
		assert.False(t, result.AllNodesCompleted, "workflow should not be complete when not all nodes are completed (legacy behavior)")
	})

	t.Run("Nil EndNodeTemplateID In Config - Falls Back To All Nodes", func(t *testing.T) {
		nodeAID := uuid.New()
		consignmentID := uuid.New()

		nodeA := &model.WorkflowNode{
			BaseModel:     model.BaseModel{ID: nodeAID},
			ConsignmentID: &consignmentID,
			State:         model.WorkflowNodeStateInProgress,
		}

		updateReq := &model.UpdateWorkflowNodeDTO{}

		mockRepo.On("GetWorkflowNodesByConsignmentIDInTx", ctx, (*gorm.DB)(nil), consignmentID).Return([]model.WorkflowNode{*nodeA}, nil).Once()
		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.AnythingOfType("[]model.WorkflowNode")).Return(nil).Once()

		completionConfig := &WorkflowCompletionConfig{
			EndNodeTemplateID: nil, // Explicitly nil
		}

		result, err := sm.TransitionToCompleted(ctx, nil, nodeA, updateReq, completionConfig)
		assert.NoError(t, err)
		assert.True(t, result.AllNodesCompleted, "single node completed = all nodes completed (legacy behavior)")
	})
}

func TestInitializeNodesWithUnlockConfiguration(t *testing.T) {
	mockRepo := new(MockWorkflowNodeRepository)
	sm := NewWorkflowNodeStateMachine(mockRepo)
	ctx := context.Background()

	t.Run("Resolve UnlockConfiguration From Template To Instance IDs", func(t *testing.T) {
		template1ID := uuid.New()
		template2ID := uuid.New()

		templates := []model.WorkflowNodeTemplate{
			{
				BaseModel: model.BaseModel{ID: template1ID},
			},
			{
				BaseModel: model.BaseModel{ID: template2ID},
				DependsOn: model.UUIDArray{template1ID},
				UnlockConfiguration: &model.UnlockConfig{
					AnyOf: []model.UnlockGroup{
						{
							AllOf: []model.UnlockCondition{
								{NodeTemplateID: template1ID, Outcome: strPtr("APPROVED")},
							},
						},
					},
				},
			},
		}

		parentRef := ParentRef{
			ConsignmentID: &uuid.UUID{},
		}

		node1ID := uuid.New()
		node2ID := uuid.New()

		mockRepo.On("CreateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			return len(nodes) == 2
		})).Return([]model.WorkflowNode{
			{
				BaseModel:              model.BaseModel{ID: node1ID},
				WorkflowNodeTemplateID: template1ID,
				State:                  model.WorkflowNodeStateLocked,
			},
			{
				BaseModel:              model.BaseModel{ID: node2ID},
				WorkflowNodeTemplateID: template2ID,
				State:                  model.WorkflowNodeStateLocked,
			},
		}, nil).Once()

		mockRepo.On("UpdateWorkflowNodesInTx", ctx, (*gorm.DB)(nil), mock.MatchedBy(func(nodes []model.WorkflowNode) bool {
			// Both nodes should be updated
			if len(nodes) != 2 {
				return false
			}
			// Find the node with unlock config and verify resolution
			for _, n := range nodes {
				if n.WorkflowNodeTemplateID == template2ID {
					if n.UnlockConfiguration == nil {
						return false
					}
					// The unlock config should have the node instance ID, not template ID
					if n.UnlockConfiguration.AnyOf[0].AllOf[0].NodeTemplateID != node1ID {
						return false
					}
				}
			}
			return true
		})).Return(nil).Once()

		createdNodes, newReadyNodes, err := sm.InitializeNodesFromTemplates(ctx, nil, parentRef, templates)
		assert.NoError(t, err)
		assert.Len(t, createdNodes, 2)
		assert.Len(t, newReadyNodes, 1)

		// Node 1 (no deps) should be READY
		assert.Equal(t, template1ID, newReadyNodes[0].WorkflowNodeTemplateID)

		// Verify the node with unlock config was resolved
		var node2 model.WorkflowNode
		for _, n := range createdNodes {
			if n.WorkflowNodeTemplateID == template2ID {
				node2 = n
				break
			}
		}
		assert.NotNil(t, node2.UnlockConfiguration)
	})
}
