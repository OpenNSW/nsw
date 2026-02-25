package service

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

const DEFAULT_END_NODE_TEMPLATE_ID = "e1a00001-0001-4000-b000-000000000006"

// StateTransitionResult represents the result of a workflow node state transition.
type StateTransitionResult struct {
	// UpdatedNodes contains all nodes that were updated during the transition.
	UpdatedNodes []model.WorkflowNode

	// NewReadyNodes contains nodes that transitioned from LOCKED to READY.
	NewReadyNodes []model.WorkflowNode

	// WorkflowFinished indicates whether all requirement met to Finish a consignment
	WorkflowFinished bool
}

// WorkflowCompletionConfig holds configuration for determining workflow completion.
// If EndNodeID is set, the workflow is complete when that specific end node is completed.
// If EndNodeID is nil, the workflow is complete when all nodes are completed (backward compatible).
type WorkflowCompletionConfig struct {
	EndNodeID *uuid.UUID
}

// ParentRef identifies the parent entity (consignment or pre-consignment) that owns workflow nodes.
// Exactly one of ConsignmentID or PreConsignmentID must be set.
type ParentRef struct {
	ConsignmentID    *uuid.UUID
	PreConsignmentID *uuid.UUID
}

// WorkflowNodeStateMachine handles workflow node state transitions and dependency propagation.
// It encapsulates the business logic for transitioning nodes between states and
// automatically unlocking dependent nodes when their dependencies are satisfied.
type WorkflowNodeStateMachine struct {
	nodeRepo WorkflowNodeRepository
}

// NewWorkflowNodeStateMachine creates a new instance of WorkflowNodeStateMachine.
func NewWorkflowNodeStateMachine(nodeRepo WorkflowNodeRepository) *WorkflowNodeStateMachine {
	return &WorkflowNodeStateMachine{
		nodeRepo: nodeRepo,
	}
}

// TransitionToCompleted transitions a workflow node to COMPLETED state and propagates
// the change to dependent nodes, unlocking them if all their dependencies are met.
// Returns a StateTransitionResult containing all updated nodes and newly ready nodes.
// The completionConfig determines how workflow completion is evaluated.
func (sm *WorkflowNodeStateMachine) TransitionToCompleted(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
	completionConfig ...*WorkflowCompletionConfig,
) (*StateTransitionResult, error) {
	if node == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}

	if node.State == model.WorkflowNodeStateCompleted {
		// Already completed, no transition needed
		return &StateTransitionResult{
			UpdatedNodes:     []model.WorkflowNode{},
			NewReadyNodes:    []model.WorkflowNode{},
			WorkflowFinished: false,
		}, nil
	}

	if !sm.canTransitionToCompleted(node.State) {
		return nil, fmt.Errorf("cannot transition node %s from state %s to COMPLETED", node.ID, node.State)
	}

	// Update the current node to COMPLETED
	node.State = model.WorkflowNodeStateCompleted
	node.ExtendedState = updateReq.ExtendedState
	node.Outcome = updateReq.Outcome
	nodesToUpdate := []model.WorkflowNode{*node}
	updatedNodeIndex := map[uuid.UUID]int{node.ID: 0}

	// Get all sibling nodes to check dependencies
	allNodes, err := sm.getSiblingNodes(ctx, tx, node)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sibling workflow nodes: %w", err)
	}

	// Extract completion config if provided
	var cfg *WorkflowCompletionConfig
	if len(completionConfig) > 0 {
		cfg = completionConfig[0]
	}

	// Build a shared node state map for unlock and completion evaluation.
	nodeStateMap := sm.buildNodeStateMap(allNodes)
	nodeStateMap[node.ID] = *node

	// Find and unlock dependent nodes.
	unlockedNodes := sm.unlockDependentNodes(allNodes, nodeStateMap)
	for _, unlockedNode := range unlockedNodes {
		if _, exists := updatedNodeIndex[unlockedNode.ID]; !exists {
			nodesToUpdate = append(nodesToUpdate, unlockedNode)
			updatedNodeIndex[unlockedNode.ID] = len(nodesToUpdate) - 1
		}
	}

	// Auto-complete the end node when it becomes READY.
	if cfg != nil && cfg.EndNodeID != nil {
		endNodeID := *cfg.EndNodeID
		if endNode, exists := nodeStateMap[endNodeID]; exists && endNode.State == model.WorkflowNodeStateReady {
			endNode.State = model.WorkflowNodeStateCompleted
			nodeStateMap[endNodeID] = endNode
			if index, found := updatedNodeIndex[endNodeID]; found {
				nodesToUpdate[index] = endNode
			} else {
				nodesToUpdate = append(nodesToUpdate, endNode)
				updatedNodeIndex[endNodeID] = len(nodesToUpdate) - 1
			}
		}
	}

	newReadyNodes := make([]model.WorkflowNode, 0, len(unlockedNodes))
	for _, unlockedNode := range unlockedNodes {
		if nodeStateMap[unlockedNode.ID].State == model.WorkflowNodeStateReady {
			newReadyNodes = append(newReadyNodes, unlockedNode)
		}
	}

	// Sort nodes by ID to prevent deadlocks
	sm.sortNodesByID(nodesToUpdate)

	// Check if workflow is completed
	allCompleted := sm.evaluateWorkflowCompletion(allNodes, nodeStateMap, cfg)

	// Persist the updates
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, nodesToUpdate); err != nil {
		return nil, fmt.Errorf("failed to update workflow nodes: %w", err)
	}

	return &StateTransitionResult{
		UpdatedNodes:     nodesToUpdate,
		NewReadyNodes:    newReadyNodes,
		WorkflowFinished: allCompleted,
	}, nil
}

// TransitionToFailed transitions a workflow node to FAILED state.
// This is a terminal state that does not propagate to dependent nodes.
func (sm *WorkflowNodeStateMachine) TransitionToFailed(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if node.State == model.WorkflowNodeStateFailed {
		// Already failed, no transition needed
		return nil
	}

	if !sm.canTransitionToFailed(node.State) {
		return fmt.Errorf("cannot transition node %s from state %s to FAILED", node.ID, node.State)
	}

	node.State = model.WorkflowNodeStateFailed
	node.ExtendedState = updateReq.ExtendedState
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, []model.WorkflowNode{*node}); err != nil {
		return fmt.Errorf("failed to update workflow node %s to FAILED state: %w", node.ID, err)
	}

	return nil
}

// TransitionToInProgress transitions a workflow node to IN_PROGRESS state.
// This indicates that work on the node has started, and it is in some intermediate state before completion.
func (sm *WorkflowNodeStateMachine) TransitionToInProgress(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if updateReq.ExtendedState == node.ExtendedState && node.State == model.WorkflowNodeStateInProgress {
		// No state change needed if already IN_PROGRESS with the same extended state
		return nil
	} else if !sm.canTransitionToInProgress(node.State) {
		return fmt.Errorf("cannot transition node %s from state %s to IN_PROGRESS", node.ID, node.State)
	} else {
		node.State = model.WorkflowNodeStateInProgress
	}
	node.ExtendedState = updateReq.ExtendedState
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, []model.WorkflowNode{*node}); err != nil {
		return fmt.Errorf("failed to update workflow node %s to IN_PROGRESS state: %w", node.ID, err)
	}

	return nil
}

// InitializeNodesFromTemplates creates workflow nodes from templates and sets up their dependencies.
// Nodes without dependencies are automatically set to READY state.
// The parentRef determines whether nodes belong to a consignment or pre-consignment.
func (sm *WorkflowNodeStateMachine) InitializeNodesFromTemplates(
	ctx context.Context,
	tx *gorm.DB,
	parentRef ParentRef,
	nodeTemplates []model.WorkflowNodeTemplate,
	workflowTemplates []model.WorkflowTemplate,
) ([]model.WorkflowNode, []model.WorkflowNode, *model.WorkflowNode, error) {
	if len(nodeTemplates) == 0 {
		return []model.WorkflowNode{}, []model.WorkflowNode{}, nil, nil
	}

	// Build lookup maps for efficient dependency resolution
	templateMap := make(map[uuid.UUID]model.WorkflowNodeTemplate)
	for _, t := range nodeTemplates {
		templateMap[t.ID] = t
	}

	// Create initial nodes in LOCKED state
	workflowNodes := make([]model.WorkflowNode, 0, len(nodeTemplates))
	for _, template := range nodeTemplates {
		workflowNode := model.WorkflowNode{
			ConsignmentID:          parentRef.ConsignmentID,
			PreConsignmentID:       parentRef.PreConsignmentID,
			WorkflowNodeTemplateID: template.ID,
			State:                  model.WorkflowNodeStateLocked,
			DependsOn:              model.UUIDArray(make([]uuid.UUID, 0)),
		}
		workflowNodes = append(workflowNodes, workflowNode)
	}

	// Create UUIDArray of all DepEndNodeTemplateIDs from workflow templates
	depEndNodeTemplateIDs := model.UUIDArray(make([]uuid.UUID, 0))
	for _, wt := range workflowTemplates {
		if wt.EndNodeTemplateID != nil {
			depEndNodeTemplateIDs = append(depEndNodeTemplateIDs, *wt.EndNodeTemplateID)
		}
	}

	// Add a endNode if parent ref is consignment (not pre-consignment)
	if parentRef.ConsignmentID != nil && len(depEndNodeTemplateIDs) > 0 {
		endNode := model.WorkflowNode{
			ConsignmentID:          parentRef.ConsignmentID,
			PreConsignmentID:       parentRef.PreConsignmentID,
			WorkflowNodeTemplateID: uuid.MustParse(DEFAULT_END_NODE_TEMPLATE_ID), // Use the default end node template ID
			State:                  model.WorkflowNodeStateLocked,
		}
		workflowNodes = append(workflowNodes, endNode)
	}

	// Persist nodes to get their IDs
	createdNodes, err := sm.nodeRepo.CreateWorkflowNodesInTx(ctx, tx, workflowNodes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create workflow nodes: %w", err)
	}

	nodeByTemplateID := make(map[uuid.UUID]model.WorkflowNode)
	for _, node := range createdNodes {
		nodeByTemplateID[node.WorkflowNodeTemplateID] = node
	}

	// Resolve dependencies from template IDs to node IDs and collect nodes that need updates
	var nodesToUpdate []model.WorkflowNode
	var newReadyNodes []model.WorkflowNode

	// Build template ID -> node instance ID mapping for UnlockConfiguration resolution
	// Note: Within a workflow, each template ID should correspond to exactly one node instance, so this mapping is valid.
	templateToNodeID := make(map[uuid.UUID]uuid.UUID)
	for templateID, node := range nodeByTemplateID {
		templateToNodeID[templateID] = node.ID
	}

	var endNode_ *model.WorkflowNode
	for i, node := range createdNodes {
		template, exists := templateMap[node.WorkflowNodeTemplateID]
		if !exists {
			// If node is the end node (which has no template), skip dependency resolution
			if node.WorkflowNodeTemplateID == uuid.MustParse(DEFAULT_END_NODE_TEMPLATE_ID) {
				dependsOnNodeID := make([]uuid.UUID, 0)
				for _, endNodeTemplateID := range depEndNodeTemplateIDs {
					if depNode, found := nodeByTemplateID[endNodeTemplateID]; found {
						dependsOnNodeID = append(dependsOnNodeID, depNode.ID)
					}
				}
				createdNodes[i].DependsOn = dependsOnNodeID
				nodesToUpdate = append(nodesToUpdate, createdNodes[i])
				endNode_ = &createdNodes[i]
				continue
			}
			return nil, nil, nil, fmt.Errorf("workflow node template with ID %s not found", node.WorkflowNodeTemplateID)
		}

		dependsOnNodeIDs := make([]uuid.UUID, 0)
		for _, dependsOnTemplateID := range template.DependsOn {
			if depNode, found := nodeByTemplateID[dependsOnTemplateID]; found {
				dependsOnNodeIDs = append(dependsOnNodeIDs, depNode.ID)
			}
		}
		createdNodes[i].DependsOn = dependsOnNodeIDs

		// Resolve UnlockConfiguration from template-level (template IDs) to instance-level (node IDs)
		if template.UnlockConfiguration != nil {
			resolvedConfig, err := template.UnlockConfiguration.ResolveToInstanceIDs(templateToNodeID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to resolve unlock configuration for node template %s: %w", template.ID, err)
			}
			createdNodes[i].UnlockConfiguration = resolvedConfig
		}

		// Determine if this node needs to be updated
		needsUpdate := false

		// Node needs update if it has dependencies
		if len(dependsOnNodeIDs) > 0 {
			needsUpdate = true
		}

		// Node needs update if it has an unlock configuration
		if createdNodes[i].UnlockConfiguration != nil {
			needsUpdate = true
		}

		// Node needs update if it has no dependencies and no unlock config (will be set to READY)
		if len(dependsOnNodeIDs) == 0 && createdNodes[i].UnlockConfiguration == nil {
			createdNodes[i].State = model.WorkflowNodeStateReady
			newReadyNodes = append(newReadyNodes, createdNodes[i])
			needsUpdate = true
		}

		if needsUpdate {
			nodesToUpdate = append(nodesToUpdate, createdNodes[i])
		}
	}

	// Persist updates only for nodes that changed
	if len(nodesToUpdate) > 0 {
		if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, nodesToUpdate); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update workflow nodes with dependencies: %w", err)
		}
	}

	return createdNodes, newReadyNodes, endNode_, nil
}

// unlockDependentNodes finds all locked nodes whose dependencies are now met and unlocks them.
// Returns the nodes that were unlocked in this transition.
func (sm *WorkflowNodeStateMachine) unlockDependentNodes(
	allNodes []model.WorkflowNode,
	nodeStateMap map[uuid.UUID]model.WorkflowNode,
) []model.WorkflowNode {
	var unlockedNodes []model.WorkflowNode

	// Check each locked node to see if its dependencies are now met
	for _, node := range allNodes {
		if node.State != model.WorkflowNodeStateLocked {
			continue
		}

		if sm.areDependenciesMet(node, nodeStateMap) {
			node.State = model.WorkflowNodeStateReady
			unlockedNodes = append(unlockedNodes, node)
			nodeStateMap[node.ID] = node
		}
	}

	return unlockedNodes
}

// areDependenciesMet checks if all dependencies for a node are satisfied.
// If the node has an UnlockConfiguration, it evaluates its boolean expression.
// Otherwise, it uses the legacy AND-all logic on the DependsOn list.
func (sm *WorkflowNodeStateMachine) areDependenciesMet(
	node model.WorkflowNode,
	nodeMap map[uuid.UUID]model.WorkflowNode,
) bool {
	// If the node has a conditional unlock configuration, use boolean expression evaluation
	if node.UnlockConfiguration != nil {
		return node.UnlockConfiguration.Evaluate(nodeMap)
	}

	// Legacy behavior: all dependencies must be COMPLETED (AND-all)
	for _, depID := range node.DependsOn {
		depNode, exists := nodeMap[depID]
		if !exists {
			return false
		}
		if depNode.State != model.WorkflowNodeStateCompleted {
			return false
		}
	}
	return true
}

// evaluateWorkflowCompletion checks if the workflow is completed.
// If a WorkflowCompletionConfig with an EndNodeID is provided, the workflow
// is considered complete when the end node is COMPLETED.
// Otherwise, falls back to checking if ALL nodes are COMPLETED (backward compatible).
func (sm *WorkflowNodeStateMachine) evaluateWorkflowCompletion(
	allNodes []model.WorkflowNode,
	nodeStateMap map[uuid.UUID]model.WorkflowNode,
	config *WorkflowCompletionConfig,
) bool {

	// If an end node ID is configured, check only that specific node
	if config != nil && config.EndNodeID != nil {
		endNodeID := *config.EndNodeID
		endNode, exists := nodeStateMap[endNodeID]
		if !exists {
			return false
		}
		return endNode.State == model.WorkflowNodeStateCompleted
	}

	// Legacy behavior: all nodes must be COMPLETED
	for _, node := range allNodes {
		state := node.State
		if current, exists := nodeStateMap[node.ID]; exists {
			state = current.State
		}
		if state != model.WorkflowNodeStateCompleted {
			return false
		}
	}

	return true
}

// buildNodeStateMap builds a node state map keyed by node ID.
func (sm *WorkflowNodeStateMachine) buildNodeStateMap(allNodes []model.WorkflowNode) map[uuid.UUID]model.WorkflowNode {
	nodeStateMap := make(map[uuid.UUID]model.WorkflowNode, len(allNodes))
	for _, node := range allNodes {
		nodeStateMap[node.ID] = node
	}
	return nodeStateMap
}

// canTransitionToCompleted checks if a node can transition to COMPLETED from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToCompleted(currentState model.WorkflowNodeState) bool {
	// Only READY or IN_PROGRESS nodes can be completed
	return currentState == model.WorkflowNodeStateReady ||
		currentState == model.WorkflowNodeStateInProgress
}

// canTransitionToFailed checks if a node can transition to FAILED from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToFailed(currentState model.WorkflowNodeState) bool {
	// Only READY or IN_PROGRESS nodes can be completed
	return currentState == model.WorkflowNodeStateReady ||
		currentState == model.WorkflowNodeStateInProgress
}

// canTransitionToInProgress checks if a node can transition to IN_PROGRESS from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToInProgress(currentState model.WorkflowNodeState) bool {
	// Only READY or FAILED nodes can be moved to IN_PROGRESS
	return currentState == model.WorkflowNodeStateReady ||
		currentState == model.WorkflowNodeStateFailed
}

// sortNodesByID sorts workflow nodes by ID to ensure consistent ordering and prevent deadlocks.
// Uses Go's standard library sort for O(n log n) performance.
func (sm *WorkflowNodeStateMachine) sortNodesByID(nodes []model.WorkflowNode) {
	sort.Slice(nodes, func(i, j int) bool {
		// Compare UUIDs directly as byte arrays for better performance
		return bytes.Compare(nodes[i].ID[:], nodes[j].ID[:]) < 0
	})
}

// getSiblingNodes retrieves all workflow nodes that share the same parent (consignment or pre-consignment).
func (sm *WorkflowNodeStateMachine) getSiblingNodes(ctx context.Context, tx *gorm.DB, node *model.WorkflowNode) ([]model.WorkflowNode, error) {
	if node.ConsignmentID != nil {
		return sm.nodeRepo.GetWorkflowNodesByConsignmentIDInTx(ctx, tx, *node.ConsignmentID)
	}
	if node.PreConsignmentID != nil {
		return sm.nodeRepo.GetWorkflowNodesByPreConsignmentIDInTx(ctx, tx, *node.PreConsignmentID)
	}
	return nil, fmt.Errorf("workflow node %s has neither consignment nor pre-consignment parent", node.ID)
}
