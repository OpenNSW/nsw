package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// StateTransitionResult represents the result of a workflow node state transition.
type StateTransitionResult struct {
	// UpdatedNodes contains all nodes that were updated during the transition.
	UpdatedNodes []model.WorkflowNode

	// NewReadyNodes contains nodes that transitioned from LOCKED to READY.
	NewReadyNodes []model.WorkflowNode

	// AllNodesCompleted indicates whether all nodes in the consignment are now completed.
	AllNodesCompleted bool
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
func (sm *WorkflowNodeStateMachine) TransitionToCompleted(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
) (*StateTransitionResult, error) {
	if node == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}

	if node.State == model.WorkflowNodeStateCompleted {
		// Already completed, no transition needed
		return &StateTransitionResult{
			UpdatedNodes:      []model.WorkflowNode{},
			NewReadyNodes:     []model.WorkflowNode{},
			AllNodesCompleted: false,
		}, nil
	}

	if !sm.canTransitionToCompleted(node.State) {
		return nil, fmt.Errorf("cannot transition node %s from state %s to COMPLETED", node.ID, node.State)
	}

	// Update the current node to COMPLETED
	node.State = model.WorkflowNodeStateCompleted
	nodesToUpdate := []model.WorkflowNode{*node}

	// Get all nodes for this consignment to check dependencies
	allNodes, err := sm.nodeRepo.GetWorkflowNodesByConsignmentIDInTx(ctx, tx, node.ConsignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow nodes for consignment %s: %w", node.ConsignmentID, err)
	}

	// Find and unlock dependent nodes
	newReadyNodes, unlockedNodes := sm.unlockDependentNodes(allNodes, node.ID)
	nodesToUpdate = append(nodesToUpdate, unlockedNodes...)

	// Sort nodes by ID to prevent deadlocks
	sm.sortNodesByID(nodesToUpdate)

	// Persist the updates
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, nodesToUpdate); err != nil {
		return nil, fmt.Errorf("failed to update workflow nodes for consignment %s: %w", node.ConsignmentID, err)
	}

	// Check if all nodes are completed
	allCompleted := sm.areAllNodesCompleted(allNodes, nodesToUpdate)

	return &StateTransitionResult{
		UpdatedNodes:      nodesToUpdate,
		NewReadyNodes:     newReadyNodes,
		AllNodesCompleted: allCompleted,
	}, nil
}

// TransitionToFailed transitions a workflow node to FAILED state.
// This is a terminal state that does not propagate to dependent nodes.
func (sm *WorkflowNodeStateMachine) TransitionToFailed(
	ctx context.Context,
	tx *gorm.DB,
	node *model.WorkflowNode,
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
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, []model.WorkflowNode{*node}); err != nil {
		return fmt.Errorf("failed to update workflow node %s to FAILED state: %w", node.ID, err)
	}

	return nil
}

// InitializeNodesFromTemplates creates workflow nodes from templates and sets up their dependencies.
// Nodes without dependencies are automatically set to READY state.
func (sm *WorkflowNodeStateMachine) InitializeNodesFromTemplates(
	ctx context.Context,
	tx *gorm.DB,
	consignmentID uuid.UUID,
	nodeTemplates []model.WorkflowNodeTemplate,
) ([]model.WorkflowNode, []model.WorkflowNode, error) {
	if len(nodeTemplates) == 0 {
		return []model.WorkflowNode{}, []model.WorkflowNode{}, nil
	}

	// Create initial nodes in LOCKED state
	workflowNodes := make([]model.WorkflowNode, 0, len(nodeTemplates))
	for _, template := range nodeTemplates {
		workflowNode := model.WorkflowNode{
			ConsignmentID:          consignmentID,
			WorkflowNodeTemplateID: template.ID,
			State:                  model.WorkflowNodeStateLocked,
			DependsOn:              model.UUIDArray(make([]uuid.UUID, 0)),
		}
		workflowNodes = append(workflowNodes, workflowNode)
	}

	// Persist nodes to get their IDs
	createdNodes, err := sm.nodeRepo.CreateWorkflowNodesInTx(ctx, tx, workflowNodes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create workflow nodes: %w", err)
	}

	// Build lookup maps for efficient dependency resolution
	templateMap := make(map[uuid.UUID]model.WorkflowNodeTemplate)
	for _, t := range nodeTemplates {
		templateMap[t.ID] = t
	}

	nodeByTemplateID := make(map[uuid.UUID]model.WorkflowNode)
	for _, node := range createdNodes {
		nodeByTemplateID[node.WorkflowNodeTemplateID] = node
	}

	// Resolve dependencies from template IDs to node IDs and collect nodes that need updates
	var nodesToUpdate []model.WorkflowNode
	var newReadyNodes []model.WorkflowNode

	for i, node := range createdNodes {
		template, exists := templateMap[node.WorkflowNodeTemplateID]
		if !exists {
			return nil, nil, fmt.Errorf("workflow node template with ID %s not found", node.WorkflowNodeTemplateID)
		}

		// Initialize as empty model.UUIDArray to avoid nil assignment
		dependsOnNodeIDs := model.UUIDArray(make([]uuid.UUID, 0))
		for _, dependsOnTemplateID := range template.DependsOn {
			if depNode, found := nodeByTemplateID[dependsOnTemplateID]; found {
				dependsOnNodeIDs = append(dependsOnNodeIDs, depNode.ID)
			}
		}
		createdNodes[i].DependsOn = dependsOnNodeIDs

		// Determine if this node needs to be updated
		needsUpdate := false

		// Node needs update if it has dependencies
		if len(dependsOnNodeIDs) > 0 {
			needsUpdate = true
		}

		// Node needs update if it has no dependencies (will be set to READY)
		if len(dependsOnNodeIDs) == 0 {
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
			return nil, nil, fmt.Errorf("failed to update workflow nodes with dependencies: %w", err)
		}
	}

	return createdNodes, newReadyNodes, nil
}

// unlockDependentNodes finds all locked nodes whose dependencies are now met and unlocks them.
// Returns both the newly ready nodes and all nodes that need to be updated.
func (sm *WorkflowNodeStateMachine) unlockDependentNodes(
	allNodes []model.WorkflowNode,
	completedNodeID uuid.UUID,
) ([]model.WorkflowNode, []model.WorkflowNode) {
	// Build a map of current node states, including the newly completed node
	nodeMap := make(map[uuid.UUID]model.WorkflowNode)
	for _, node := range allNodes {
		if node.ID == completedNodeID {
			node.State = model.WorkflowNodeStateCompleted
		}
		nodeMap[node.ID] = node
	}

	var newReadyNodes []model.WorkflowNode
	var unlockedNodes []model.WorkflowNode

	// Check each locked node to see if its dependencies are now met
	for _, node := range allNodes {
		if node.State != model.WorkflowNodeStateLocked {
			continue
		}

		if sm.areDependenciesMet(node.DependsOn, nodeMap) {
			node.State = model.WorkflowNodeStateReady
			newReadyNodes = append(newReadyNodes, node)
			unlockedNodes = append(unlockedNodes, node)
		}
	}

	return newReadyNodes, unlockedNodes
}

// areDependenciesMet checks if all dependencies for a node are in COMPLETED state.
func (sm *WorkflowNodeStateMachine) areDependenciesMet(
	dependsOn []uuid.UUID,
	nodeMap map[uuid.UUID]model.WorkflowNode,
) bool {
	for _, depID := range dependsOn {
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

// areAllNodesCompleted checks if all nodes are in COMPLETED state, considering pending updates.
func (sm *WorkflowNodeStateMachine) areAllNodesCompleted(
	allNodes []model.WorkflowNode,
	updatedNodes []model.WorkflowNode,
) bool {
	// Build map of updated states
	updatedStateMap := make(map[uuid.UUID]model.WorkflowNodeState)
	for _, node := range updatedNodes {
		updatedStateMap[node.ID] = node.State
	}

	// Check all nodes
	for _, node := range allNodes {
		state := node.State
		if updatedState, wasUpdated := updatedStateMap[node.ID]; wasUpdated {
			state = updatedState
		}
		if state != model.WorkflowNodeStateCompleted {
			return false
		}
	}

	return true
}

// canTransitionToCompleted checks if a node can transition to COMPLETED from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToCompleted(currentState model.WorkflowNodeState) bool {
	// Only READY or IN_PROGRESS nodes can be completed
	return currentState == model.WorkflowNodeStateReady ||
		currentState == model.WorkflowNodeStateInProgress
}

// canTransitionToFailed checks if a node can transition to FAILED from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToFailed(currentState model.WorkflowNodeState) bool {
	// Any non-terminal state can transition to FAILED
	return currentState != model.WorkflowNodeStateFailed &&
		currentState != model.WorkflowNodeStateCompleted
}

// sortNodesByID sorts workflow nodes by ID to ensure consistent ordering and prevent deadlocks.
func (sm *WorkflowNodeStateMachine) sortNodesByID(nodes []model.WorkflowNode) {
	n := len(nodes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if nodes[j].ID.String() > nodes[j+1].ID.String() {
				nodes[j], nodes[j+1] = nodes[j+1], nodes[j]
			}
		}
	}
}
