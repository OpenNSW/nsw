package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UnlockCondition represents a single condition that checks a specific dependency node's state and/or outcome.
// Both State and Outcome are optional.
// When both are specified, they are combined with AND logic (both must match).
// Used within UnlockGroup to form AND conditions across multiple nodes.
type UnlockCondition struct {
	// NodeTemplateID references the workflow node template to check (template-level).
	NodeTemplateID uuid.UUID `json:"nodeTemplateId"`

	// NodeID is the resolved workflow node instance ID (after resolution). Optional in the condition definition, but will be populated during evaluation.
	NodeID *uuid.UUID `json:"nodeId,omitempty"`

	// State is the expected state of the referenced node (e.g., "COMPLETED", "FAILED").
	// Optional — if nil, the node's state is not checked.
	State *string `json:"state,omitempty"`

	// Outcome is the expected outcome value of the referenced node (e.g., "APPROVED", "REJECTED").
	// Optional — if nil, the node's outcome is not checked.
	Outcome *string `json:"outcome,omitempty"`
}

// UnlockGroup represents a group of conditions that must ALL be true (AND logic).
// Multiple UnlockGroups within an UnlockConfig form OR logic.
type UnlockGroup struct {
	AllOf []UnlockCondition `json:"allOf"`
}

// UnlockExpression represents a recursive boolean expression for unlock logic.
// Exactly one of the following must be present:
//   - AnyOf: OR across child expressions
//   - AllOf: AND across child expressions
//   - Leaf condition: NodeTemplateID (with optional State and/or Outcome)
//
// This enables arbitrary nesting of AND/OR expressions.
type UnlockExpression struct {
	AnyOf []UnlockExpression `json:"anyOf,omitempty"`
	AllOf []UnlockExpression `json:"allOf,omitempty"`

	NodeTemplateID uuid.UUID  `json:"nodeTemplateId,omitempty"`
	NodeID         *uuid.UUID `json:"nodeId,omitempty"`
	State          *string    `json:"state,omitempty"`
	Outcome        *string    `json:"outcome,omitempty"`
}

// UnlockConfig represents the unlock configuration for a workflow node.
// It supports two formats:
//   - Legacy DNF format via AnyOf (OR of AND groups)
//   - General recursive expression format via Expression
//
// Example JSON:
//
//	{
//	  "anyOf": [
//	    {
//	      "allOf": [
//	        {"nodeTemplateId": "uuid-1", "state": "COMPLETED", "outcome": "APPROVED"},
//	        {"nodeTemplateId": "uuid-2", "state": "COMPLETED"}
//	      ]
//	    },
//	    {
//	      "allOf": [
//	        {"nodeTemplateId": "uuid-1", "outcome": "FAST_TRACKED"}
//	      ]
//	    }
//	  ]
//	}
//
// This means: (node1.state == COMPLETED && node1.outcome == APPROVED && node2.state == COMPLETED) OR (node1.outcome == FAST_TRACKED)
type UnlockConfig struct {
	// Legacy format: DNF (OR of AND groups).
	AnyOf []UnlockGroup `json:"anyOf,omitempty"`

	// General format: recursive AND/OR expression.
	// Example:
	// {
	//   "expression": {
	//     "allOf": [
	//       {"nodeTemplateId": "uuid-1", "state": "COMPLETED"},
	//       {
	//         "anyOf": [
	//           {"nodeTemplateId": "uuid-2", "outcome": "APPROVED"},
	//           {"nodeTemplateId": "uuid-3", "state": "FAILED"}
	//         ]
	//       }
	//     ]
	//   }
	// }
	Expression *UnlockExpression `json:"expression,omitempty"`
}

// Validate checks that the unlock configuration is well-formed.
func (uc *UnlockConfig) Validate() error {
	if uc.Expression != nil && len(uc.AnyOf) > 0 {
		return fmt.Errorf("unlock configuration cannot define both expression and anyOf")
	}

	if uc.Expression != nil {
		return uc.validateExpression(*uc.Expression, "expression")
	}

	if len(uc.AnyOf) == 0 {
		return fmt.Errorf("unlock configuration must have at least one group in anyOf or define expression")
	}
	for i, group := range uc.AnyOf {
		if len(group.AllOf) == 0 {
			return fmt.Errorf("unlock configuration group %d must have at least one condition in allOf", i)
		}
		for j, cond := range group.AllOf {
			if err := uc.validateCondition(cond, fmt.Sprintf("unlock configuration group %d condition %d", i, j)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (uc *UnlockConfig) validateCondition(cond UnlockCondition, path string) error {
	if cond.NodeTemplateID == uuid.Nil {
		return fmt.Errorf("%s has nil nodeTemplateId", path)
	}
	if cond.State == nil && cond.Outcome == nil {
		return fmt.Errorf("%s must specify at least state or outcome", path)
	}
	if cond.State != nil && len(strings.TrimSpace(*cond.State)) == 0 {
		return fmt.Errorf("%s has empty state", path)
	}
	if cond.Outcome != nil && len(strings.TrimSpace(*cond.Outcome)) == 0 {
		return fmt.Errorf("%s has empty outcome", path)
	}
	return nil
}

func (uc *UnlockConfig) validateExpression(expr UnlockExpression, path string) error {
	hasAny := len(expr.AnyOf) > 0
	hasAll := len(expr.AllOf) > 0
	hasLeaf := expr.NodeTemplateID != uuid.Nil || expr.State != nil || expr.Outcome != nil

	definedCount := 0
	if hasAny {
		definedCount++
	}
	if hasAll {
		definedCount++
	}
	if hasLeaf {
		definedCount++
	}

	if definedCount == 0 {
		return fmt.Errorf("%s must define one of anyOf, allOf, or a condition", path)
	}
	if definedCount > 1 {
		return fmt.Errorf("%s must define exactly one of anyOf, allOf, or a condition", path)
	}

	if hasAny {
		for i, child := range expr.AnyOf {
			if err := uc.validateExpression(child, fmt.Sprintf("%s.anyOf[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}

	if hasAll {
		for i, child := range expr.AllOf {
			if err := uc.validateExpression(child, fmt.Sprintf("%s.allOf[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.validateCondition(UnlockCondition{
		NodeTemplateID: expr.NodeTemplateID,
		State:          expr.State,
		Outcome:        expr.Outcome,
	}, path)
}

// ResolveToInstanceIDs creates a copy of the UnlockConfig with template IDs replaced by instance node IDs.
// The templateToNodeID map should contain template ID -> node instance ID mappings.
func (uc *UnlockConfig) ResolveToInstanceIDs(templateToNodeID map[uuid.UUID]uuid.UUID) (*UnlockConfig, error) {
	// Validate the config before resolution
	if err := uc.Validate(); err != nil {
		return nil, fmt.Errorf("invalid unlock configuration: %w", err)
	}

	// If it's an expression-based config, resolve the expression
	if uc.Expression != nil {
		resolvedExpr, err := uc.resolveExpressionToInstanceIDs(*uc.Expression, templateToNodeID)
		if err != nil {
			return nil, err
		}
		return &UnlockConfig{Expression: &resolvedExpr}, nil
	}

	resolved := &UnlockConfig{
		AnyOf: make([]UnlockGroup, len(uc.AnyOf)),
	}
	for i, group := range uc.AnyOf {
		resolved.AnyOf[i] = UnlockGroup{
			AllOf: make([]UnlockCondition, len(group.AllOf)),
		}
		for j, cond := range group.AllOf {
			nodeID, found := templateToNodeID[cond.NodeTemplateID]
			if !found {
				return nil, fmt.Errorf("no instance node found for template ID %s in unlock configuration", cond.NodeTemplateID)
			}
			resolved.AnyOf[i].AllOf[j] = UnlockCondition{
				NodeTemplateID: cond.NodeTemplateID,
				NodeID:         &nodeID,
				State:          cond.State,
				Outcome:        cond.Outcome,
			}
		}
	}
	return resolved, nil
}

func (uc *UnlockConfig) resolveExpressionToInstanceIDs(expr UnlockExpression, templateToNodeID map[uuid.UUID]uuid.UUID) (UnlockExpression, error) {
	resolved := UnlockExpression{
		AnyOf:   make([]UnlockExpression, len(expr.AnyOf)),
		AllOf:   make([]UnlockExpression, len(expr.AllOf)),
		State:   expr.State,
		Outcome: expr.Outcome,
	}

	for i, child := range expr.AnyOf {
		childResolved, err := uc.resolveExpressionToInstanceIDs(child, templateToNodeID)
		if err != nil {
			return UnlockExpression{}, err
		}
		resolved.AnyOf[i] = childResolved
	}

	for i, child := range expr.AllOf {
		childResolved, err := uc.resolveExpressionToInstanceIDs(child, templateToNodeID)
		if err != nil {
			return UnlockExpression{}, err
		}
		resolved.AllOf[i] = childResolved
	}

	if expr.NodeTemplateID != uuid.Nil {
		nodeID, found := templateToNodeID[expr.NodeTemplateID]
		if !found {
			return UnlockExpression{}, fmt.Errorf("no instance node found for template ID %s in unlock configuration", expr.NodeTemplateID)
		}
		resolved.NodeID = &nodeID
		resolved.NodeTemplateID = expr.NodeTemplateID
	}

	return resolved, nil
}

// Evaluate checks if the unlock conditions are satisfied given the current node states and outcomes.
// The nodeMap should contain node ID -> WorkflowNode mappings with current states.
func (uc *UnlockConfig) Evaluate(nodeMap map[uuid.UUID]WorkflowNode) bool {
	if uc.Expression != nil {
		return uc.evaluateExpression(*uc.Expression, nodeMap)
	}

	// DNF evaluation: any group being satisfied makes the whole config satisfied (OR)
	for _, group := range uc.AnyOf {
		if uc.evaluateGroup(group, nodeMap) {
			return true
		}
	}
	return false
}

// evaluateGroup checks if all conditions in a group are satisfied (AND).
func (uc *UnlockConfig) evaluateGroup(group UnlockGroup, nodeMap map[uuid.UUID]WorkflowNode) bool {
	for _, cond := range group.AllOf {
		node, exists := nodeMap[*cond.NodeID]
		if !exists {
			return false
		}
		// Check state if specified
		if cond.State != nil && string(node.State) != *cond.State {
			return false
		}
		// Check outcome if specified
		if cond.Outcome != nil {
			if node.Outcome == nil || *node.Outcome != *cond.Outcome {
				return false
			}
		}
	}
	return true
}

func (uc *UnlockConfig) evaluateExpression(expr UnlockExpression, nodeMap map[uuid.UUID]WorkflowNode) bool {
	if len(expr.AnyOf) > 0 {
		for _, child := range expr.AnyOf {
			if uc.evaluateExpression(child, nodeMap) {
				return true
			}
		}
		return false
	}

	if len(expr.AllOf) > 0 {
		for _, child := range expr.AllOf {
			if !uc.evaluateExpression(child, nodeMap) {
				return false
			}
		}
		return true
	}

	return uc.evaluateCondition(UnlockCondition{
		NodeTemplateID: expr.NodeTemplateID,
		NodeID:         expr.NodeID,
		State:          expr.State,
		Outcome:        expr.Outcome,
	}, nodeMap)
}

func (uc *UnlockConfig) evaluateCondition(cond UnlockCondition, nodeMap map[uuid.UUID]WorkflowNode) bool {
	node, exists := nodeMap[*cond.NodeID]
	if !exists {
		return false
	}
	if cond.State != nil && string(node.State) != *cond.State {
		return false
	}
	if cond.Outcome != nil {
		if node.Outcome == nil || *node.Outcome != *cond.Outcome {
			return false
		}
	}
	return true
}

// MarshalJSON implements json.Marshaler for UnlockConfig.
func (uc UnlockConfig) MarshalJSON() ([]byte, error) {
	type Alias UnlockConfig
	return json.Marshal(Alias(uc))
}

// UnmarshalJSON implements json.Unmarshaler for UnlockConfig.
func (uc *UnlockConfig) UnmarshalJSON(data []byte) error {
	type Alias UnlockConfig
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*uc = UnlockConfig(alias)
	return nil
}
