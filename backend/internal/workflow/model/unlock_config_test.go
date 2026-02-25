package model

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func resolveForEvaluate(t *testing.T, uc *UnlockConfig, ids ...uuid.UUID) *UnlockConfig {
	t.Helper()
	templateToNodeID := make(map[uuid.UUID]uuid.UUID, len(ids))
	for _, id := range ids {
		templateToNodeID[id] = id
	}

	resolved, err := uc.ResolveToInstanceIDs(templateToNodeID)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	return resolved
}

func TestUnlockConfig_Validate(t *testing.T) {
	t.Run("Valid Config - State Only", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), State: strPtr("COMPLETED")},
					},
				},
			},
		}
		assert.NoError(t, uc.Validate())
	})

	t.Run("Valid Config - Outcome Only", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}
		assert.NoError(t, uc.Validate())
	})

	t.Run("Valid Config - State And Outcome", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}
		assert.NoError(t, uc.Validate())
	})

	t.Run("Empty AnyOf", func(t *testing.T) {
		uc := &UnlockConfig{AnyOf: []UnlockGroup{}}
		assert.Error(t, uc.Validate())
	})

	t.Run("Empty AllOf", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{AllOf: []UnlockCondition{}},
			},
		}
		assert.Error(t, uc.Validate())
	})

	t.Run("Nil NodeTemplateID", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.Nil, State: strPtr("COMPLETED")},
					},
				},
			},
		}
		assert.Error(t, uc.Validate())
	})

	t.Run("Neither State Nor Outcome", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New()},
					},
				},
			},
		}
		err := uc.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify at least state or outcome")
	})

	t.Run("Empty State String", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), State: strPtr("")},
					},
				},
			},
		}
		err := uc.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty state")
	})

	t.Run("Empty Outcome String", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), Outcome: strPtr("")},
					},
				},
			},
		}
		err := uc.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty outcome")
	})

	t.Run("Multiple Groups Valid", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
						{NodeTemplateID: uuid.New(), State: strPtr("COMPLETED")},
					},
				},
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: uuid.New(), Outcome: strPtr("FAST_TRACKED")},
					},
				},
			},
		}
		assert.NoError(t, uc.Validate())
	})
}

func TestUnlockConfig_Evaluate(t *testing.T) {
	nodeA := uuid.New()
	nodeB := uuid.New()

	t.Run("State Only - Condition Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("State Only - Wrong State", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateInProgress},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("Outcome Only - Condition Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "APPROVED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcome},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("Outcome Only - Wrong Outcome", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "REJECTED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcome},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("Outcome Only - Nil Outcome On Node", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: nil},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("State And Outcome - Both Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "APPROVED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcome},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("State And Outcome - State Met Outcome Wrong", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "REJECTED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcome},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("State And Outcome - Outcome Met State Wrong", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "APPROVED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateInProgress, Outcome: &outcome},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("Node Missing From Map", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("AND Logic - All Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
						{NodeTemplateID: nodeB, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcomeA := "APPROVED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcomeA},
			nodeB: {State: WorkflowNodeStateCompleted},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("AND Logic - Partial Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
						{NodeTemplateID: nodeB, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted},
			nodeB: {State: WorkflowNodeStateLocked},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("OR Logic - First Group Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
					},
				},
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("FAST_TRACKED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("OR Logic - Second Group Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("APPROVED")},
					},
				},
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("FAST_TRACKED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "FAST_TRACKED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcome},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("OR Logic - No Group Met", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("APPROVED")},
					},
				},
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("FAST_TRACKED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		outcome := "REJECTED"
		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted, Outcome: &outcome},
		}
		assert.False(t, resolved.Evaluate(nodeMap))
	})

	t.Run("Mixed State And Outcome In OR", func(t *testing.T) {
		// (nodeA.State == COMPLETED && nodeB.Outcome == VERIFIED) OR (nodeA.Outcome == FAST_TRACKED)
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
						{NodeTemplateID: nodeB, Outcome: strPtr("VERIFIED")},
					},
				},
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("FAST_TRACKED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		// Test case 1: First group satisfied
		outcomeB := "VERIFIED"
		nodeMap1 := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted},
			nodeB: {State: WorkflowNodeStateCompleted, Outcome: &outcomeB},
		}
		assert.True(t, resolved.Evaluate(nodeMap1))

		// Test case 2: Second group satisfied
		outcomeFT := "FAST_TRACKED"
		nodeMap2 := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateInProgress, Outcome: &outcomeFT},
			nodeB: {State: WorkflowNodeStateLocked},
		}
		assert.True(t, resolved.Evaluate(nodeMap2))

		// Test case 3: Neither group satisfied
		nodeMap3 := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted},
			nodeB: {State: WorkflowNodeStateLocked},
		}
		assert.False(t, resolved.Evaluate(nodeMap3))
	})

	t.Run("State Check FAILED", func(t *testing.T) {
		// Unlock if a dependency has FAILED
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("FAILED")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		nodeMap := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateFailed},
		}
		assert.True(t, resolved.Evaluate(nodeMap))
	})

	t.Run("OR Between State And Outcome On Same Node", func(t *testing.T) {
		// dep_node_1.State == "COMPLETED" || dep_node_1.Outcome == "X"
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
					},
				},
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeA, Outcome: strPtr("X")},
					},
				},
			},
		}

		resolved := resolveForEvaluate(t, uc, nodeA, nodeB)

		// State matches but no outcome
		nodeMap1 := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateCompleted},
		}
		assert.True(t, resolved.Evaluate(nodeMap1))

		// Outcome matches but state is not COMPLETED (e.g. IN_PROGRESS with outcome set)
		outcomeX := "X"
		nodeMap2 := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateInProgress, Outcome: &outcomeX},
		}
		assert.True(t, resolved.Evaluate(nodeMap2))

		// Neither matches
		nodeMap3 := map[uuid.UUID]WorkflowNode{
			nodeA: {State: WorkflowNodeStateLocked},
		}
		assert.False(t, resolved.Evaluate(nodeMap3))
	})
}

func TestUnlockConfig_ResolveToInstanceIDs(t *testing.T) {
	templateA := uuid.New()
	templateB := uuid.New()
	instanceA := uuid.New()
	instanceB := uuid.New()

	t.Run("Successful Resolution", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: templateA, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
						{NodeTemplateID: templateB, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		mapping := map[uuid.UUID]uuid.UUID{
			templateA: instanceA,
			templateB: instanceB,
		}

		resolved, err := uc.ResolveToInstanceIDs(mapping)
		assert.NoError(t, err)
		assert.Equal(t, templateA, resolved.AnyOf[0].AllOf[0].NodeTemplateID)
		assert.Equal(t, templateB, resolved.AnyOf[0].AllOf[1].NodeTemplateID)
		if assert.NotNil(t, resolved.AnyOf[0].AllOf[0].NodeID) {
			assert.Equal(t, instanceA, *resolved.AnyOf[0].AllOf[0].NodeID)
		}
		if assert.NotNil(t, resolved.AnyOf[0].AllOf[1].NodeID) {
			assert.Equal(t, instanceB, *resolved.AnyOf[0].AllOf[1].NodeID)
		}
		// Verify State and Outcome are preserved
		assert.Equal(t, "COMPLETED", *resolved.AnyOf[0].AllOf[0].State)
		assert.Equal(t, "APPROVED", *resolved.AnyOf[0].AllOf[0].Outcome)
		assert.Equal(t, "COMPLETED", *resolved.AnyOf[0].AllOf[1].State)
		assert.Nil(t, resolved.AnyOf[0].AllOf[1].Outcome)
	})

	t.Run("Missing Template ID", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: templateA, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		mapping := map[uuid.UUID]uuid.UUID{
			templateB: instanceB,
		}

		_, err := uc.ResolveToInstanceIDs(mapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no instance node found")
	})

	t.Run("Does Not Modify Original", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: templateA, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		mapping := map[uuid.UUID]uuid.UUID{
			templateA: instanceA,
		}

		resolved, err := uc.ResolveToInstanceIDs(mapping)
		assert.NoError(t, err)
		assert.Equal(t, templateA, uc.AnyOf[0].AllOf[0].NodeTemplateID)
		assert.Equal(t, templateA, resolved.AnyOf[0].AllOf[0].NodeTemplateID)
		if assert.NotNil(t, resolved.AnyOf[0].AllOf[0].NodeID) {
			assert.Equal(t, instanceA, *resolved.AnyOf[0].AllOf[0].NodeID)
		}
	})
}

func TestUnlockConfig_JSON(t *testing.T) {
	t.Run("Marshal And Unmarshal - State And Outcome", func(t *testing.T) {
		nodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		original := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeID, State: strPtr("COMPLETED"), Outcome: strPtr("APPROVED")},
					},
				},
			},
		}

		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var restored UnlockConfig
		err = json.Unmarshal(data, &restored)
		assert.NoError(t, err)

		assert.Equal(t, original.AnyOf[0].AllOf[0].NodeTemplateID, restored.AnyOf[0].AllOf[0].NodeTemplateID)
		assert.Equal(t, *original.AnyOf[0].AllOf[0].State, *restored.AnyOf[0].AllOf[0].State)
		assert.Equal(t, *original.AnyOf[0].AllOf[0].Outcome, *restored.AnyOf[0].AllOf[0].Outcome)
	})

	t.Run("Marshal And Unmarshal - State Only", func(t *testing.T) {
		nodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		original := &UnlockConfig{
			AnyOf: []UnlockGroup{
				{
					AllOf: []UnlockCondition{
						{NodeTemplateID: nodeID, State: strPtr("COMPLETED")},
					},
				},
			},
		}

		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var restored UnlockConfig
		err = json.Unmarshal(data, &restored)
		assert.NoError(t, err)

		assert.Equal(t, *original.AnyOf[0].AllOf[0].State, *restored.AnyOf[0].AllOf[0].State)
		assert.Nil(t, restored.AnyOf[0].AllOf[0].Outcome)
	})

	t.Run("Unmarshal From JSON String", func(t *testing.T) {
		jsonStr := `{
			"anyOf": [
				{
					"allOf": [
						{"nodeTemplateId": "00000000-0000-0000-0000-000000000001", "state": "COMPLETED", "outcome": "APPROVED"},
						{"nodeTemplateId": "00000000-0000-0000-0000-000000000002", "state": "COMPLETED"}
					]
				},
				{
					"allOf": [
						{"nodeTemplateId": "00000000-0000-0000-0000-000000000001", "outcome": "FAST_TRACKED"}
					]
				}
			]
		}`

		var uc UnlockConfig
		err := json.Unmarshal([]byte(jsonStr), &uc)
		assert.NoError(t, err)
		assert.Len(t, uc.AnyOf, 2)
		assert.Len(t, uc.AnyOf[0].AllOf, 2)
		assert.Len(t, uc.AnyOf[1].AllOf, 1)
		// First condition: state + outcome
		assert.Equal(t, "COMPLETED", *uc.AnyOf[0].AllOf[0].State)
		assert.Equal(t, "APPROVED", *uc.AnyOf[0].AllOf[0].Outcome)
		// Second condition: state only
		assert.Equal(t, "COMPLETED", *uc.AnyOf[0].AllOf[1].State)
		assert.Nil(t, uc.AnyOf[0].AllOf[1].Outcome)
		// Third condition: outcome only
		assert.Nil(t, uc.AnyOf[1].AllOf[0].State)
		assert.Equal(t, "FAST_TRACKED", *uc.AnyOf[1].AllOf[0].Outcome)
	})
}

func TestUnlockConfig_Expression_Validate(t *testing.T) {
	nodeA := uuid.New()
	nodeB := uuid.New()

	t.Run("Valid Nested Expression", func(t *testing.T) {
		uc := &UnlockConfig{
			Expression: &UnlockExpression{
				AllOf: []UnlockExpression{
					{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
					{
						AnyOf: []UnlockExpression{
							{NodeTemplateID: nodeB, Outcome: strPtr("APPROVED")},
							{NodeTemplateID: nodeB, State: strPtr("FAILED")},
						},
					},
				},
			},
		}
		assert.NoError(t, uc.Validate())
	})

	t.Run("Invalid Mixed Operator And Leaf", func(t *testing.T) {
		uc := &UnlockConfig{
			Expression: &UnlockExpression{
				NodeTemplateID: nodeA,
				AllOf: []UnlockExpression{
					{NodeTemplateID: nodeB, State: strPtr("COMPLETED")},
				},
			},
		}
		err := uc.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exactly one")
	})

	t.Run("Invalid When Both Legacy And Expression Set", func(t *testing.T) {
		uc := &UnlockConfig{
			AnyOf: []UnlockGroup{{AllOf: []UnlockCondition{{NodeTemplateID: nodeA, State: strPtr("COMPLETED")}}}},
			Expression: &UnlockExpression{
				NodeTemplateID: nodeB,
				State:          strPtr("COMPLETED"),
			},
		}
		err := uc.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot define both expression and anyOf")
	})
}

func TestUnlockConfig_Expression_Evaluate(t *testing.T) {
	nodeA := uuid.New()
	nodeB := uuid.New()
	nodeC := uuid.New()

	uc := &UnlockConfig{
		Expression: &UnlockExpression{
			AllOf: []UnlockExpression{
				{NodeTemplateID: nodeA, State: strPtr("COMPLETED")},
				{
					AnyOf: []UnlockExpression{
						{NodeTemplateID: nodeB, Outcome: strPtr("APPROVED")},
						{NodeTemplateID: nodeC, State: strPtr("FAILED")},
					},
				},
			},
		},
	}

	resolved := resolveForEvaluate(t, uc, nodeA, nodeB, nodeC)

	outcomeApproved := "APPROVED"
	assert.True(t, resolved.Evaluate(map[uuid.UUID]WorkflowNode{
		nodeA: {State: WorkflowNodeStateCompleted},
		nodeB: {State: WorkflowNodeStateCompleted, Outcome: &outcomeApproved},
		nodeC: {State: WorkflowNodeStateLocked},
	}))

	assert.True(t, resolved.Evaluate(map[uuid.UUID]WorkflowNode{
		nodeA: {State: WorkflowNodeStateCompleted},
		nodeB: {State: WorkflowNodeStateCompleted},
		nodeC: {State: WorkflowNodeStateFailed},
	}))

	assert.False(t, resolved.Evaluate(map[uuid.UUID]WorkflowNode{
		nodeA: {State: WorkflowNodeStateInProgress},
		nodeB: {State: WorkflowNodeStateCompleted, Outcome: &outcomeApproved},
		nodeC: {State: WorkflowNodeStateFailed},
	}))
}

func TestUnlockConfig_Expression_ResolveToInstanceIDs(t *testing.T) {
	templateA := uuid.New()
	templateB := uuid.New()
	instanceA := uuid.New()
	instanceB := uuid.New()

	uc := &UnlockConfig{
		Expression: &UnlockExpression{
			AllOf: []UnlockExpression{
				{NodeTemplateID: templateA, State: strPtr("COMPLETED")},
				{NodeTemplateID: templateB, Outcome: strPtr("APPROVED")},
			},
		},
	}

	resolved, err := uc.ResolveToInstanceIDs(map[uuid.UUID]uuid.UUID{
		templateA: instanceA,
		templateB: instanceB,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resolved.Expression) {
		assert.Equal(t, templateA, resolved.Expression.AllOf[0].NodeTemplateID)
		assert.Equal(t, templateB, resolved.Expression.AllOf[1].NodeTemplateID)
		if assert.NotNil(t, resolved.Expression.AllOf[0].NodeID) {
			assert.Equal(t, instanceA, *resolved.Expression.AllOf[0].NodeID)
		}
		if assert.NotNil(t, resolved.Expression.AllOf[1].NodeID) {
			assert.Equal(t, instanceB, *resolved.Expression.AllOf[1].NodeID)
		}
	}
}

func TestUnlockConfig_Expression_JSON(t *testing.T) {
	jsonStr := `{
		"expression": {
			"allOf": [
				{"nodeTemplateId": "00000000-0000-0000-0000-000000000001", "state": "COMPLETED"},
				{
					"anyOf": [
						{"nodeTemplateId": "00000000-0000-0000-0000-000000000002", "outcome": "APPROVED"},
						{"nodeTemplateId": "00000000-0000-0000-0000-000000000003", "state": "FAILED"}
					]
				}
			]
		}
	}`

	var uc UnlockConfig
	err := json.Unmarshal([]byte(jsonStr), &uc)
	assert.NoError(t, err)
	if assert.NotNil(t, uc.Expression) {
		assert.Len(t, uc.Expression.AllOf, 2)
		assert.Equal(t, "COMPLETED", *uc.Expression.AllOf[0].State)
		assert.Len(t, uc.Expression.AllOf[1].AnyOf, 2)
		assert.Equal(t, "APPROVED", *uc.Expression.AllOf[1].AnyOf[0].Outcome)
		assert.Equal(t, "FAILED", *uc.Expression.AllOf[1].AnyOf[1].State)
	}
}
