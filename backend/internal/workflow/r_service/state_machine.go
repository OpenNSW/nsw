package r_service

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/r_model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StateTransitionResult represents the result of a workflow node state transition.
type StateTransitionResult struct {
	// UpdatedNodes contains all nodes that were updated during the transition.
	UpdatedNodes []r_model.WorkflowNode

	// NewReadyNodes contains nodes that transitioned from LOCKED to READY.
	NewReadyNodes []r_model.WorkflowNode

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
	node *r_model.WorkflowNode,
) (*StateTransitionResult, error) {
	if node == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}

	if node.State == r_model.WorkflowNodeStateCompleted {
		// Already completed, no transition needed
		return &StateTransitionResult{
			UpdatedNodes:      []r_model.WorkflowNode{},
			NewReadyNodes:     []r_model.WorkflowNode{},
			AllNodesCompleted: false,
		}, nil
	}

	if !sm.canTransitionToCompleted(node.State) {
		return nil, fmt.Errorf("cannot transition node %s from state %s to COMPLETED", node.ID, node.State)
	}

	// Update the current node to COMPLETED
	node.State = r_model.WorkflowNodeStateCompleted
	nodesToUpdate := []r_model.WorkflowNode{*node}

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
	node *r_model.WorkflowNode,
) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if node.State == r_model.WorkflowNodeStateFailed {
		// Already failed, no transition needed
		return nil
	}

	if !sm.canTransitionToFailed(node.State) {
		return fmt.Errorf("cannot transition node %s from state %s to FAILED", node.ID, node.State)
	}

	node.State = r_model.WorkflowNodeStateFailed
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, []r_model.WorkflowNode{*node}); err != nil {
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
	nodeTemplates []r_model.WorkflowNodeTemplate,
) ([]r_model.WorkflowNode, []r_model.WorkflowNode, error) {
	if len(nodeTemplates) == 0 {
		return []r_model.WorkflowNode{}, []r_model.WorkflowNode{}, nil
	}

	// Create initial nodes in LOCKED state
	workflowNodes := make([]r_model.WorkflowNode, 0, len(nodeTemplates))
	for _, template := range nodeTemplates {
		workflowNode := r_model.WorkflowNode{
			ConsignmentID:          consignmentID,
			WorkflowNodeTemplateID: template.ID,
			State:                  r_model.WorkflowNodeStateLocked,
			DependsOn:              []uuid.UUID{},
		}
		workflowNodes = append(workflowNodes, workflowNode)
	}

	// Persist nodes to get their IDs
	createdNodes, err := sm.nodeRepo.CreateWorkflowNodesInTx(ctx, tx, workflowNodes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create workflow nodes: %w", err)
	}

	// Build lookup maps for efficient dependency resolution
	templateMap := make(map[uuid.UUID]r_model.WorkflowNodeTemplate)
	for _, t := range nodeTemplates {
		templateMap[t.ID] = t
	}

	nodeByTemplateID := make(map[uuid.UUID]r_model.WorkflowNode)
	for _, node := range createdNodes {
		nodeByTemplateID[node.WorkflowNodeTemplateID] = node
	}

	// Resolve dependencies from template IDs to node IDs
	for i, node := range createdNodes {
		template, exists := templateMap[node.WorkflowNodeTemplateID]
		if !exists {
			return nil, nil, fmt.Errorf("workflow node template with ID %s not found", node.WorkflowNodeTemplateID)
		}

		var dependsOnNodeIDs []uuid.UUID
		for _, dependsOnTemplateID := range template.DependsOn {
			if depNode, found := nodeByTemplateID[dependsOnTemplateID]; found {
				dependsOnNodeIDs = append(dependsOnNodeIDs, depNode.ID)
			}
		}
		createdNodes[i].DependsOn = dependsOnNodeIDs
	}

	// Set nodes without dependencies to READY
	var newReadyNodes []r_model.WorkflowNode
	for i, node := range createdNodes {
		if len(node.DependsOn) == 0 {
			createdNodes[i].State = r_model.WorkflowNodeStateReady
			newReadyNodes = append(newReadyNodes, createdNodes[i])
		}
	}

	// Persist dependency updates
	if err := sm.nodeRepo.UpdateWorkflowNodesInTx(ctx, tx, createdNodes); err != nil {
		return nil, nil, fmt.Errorf("failed to update workflow nodes with dependencies: %w", err)
	}

	return createdNodes, newReadyNodes, nil
}

// unlockDependentNodes finds all locked nodes whose dependencies are now met and unlocks them.
// Returns both the newly ready nodes and all nodes that need to be updated.
func (sm *WorkflowNodeStateMachine) unlockDependentNodes(
	allNodes []r_model.WorkflowNode,
	completedNodeID uuid.UUID,
) ([]r_model.WorkflowNode, []r_model.WorkflowNode) {
	// Build a map of current node states, including the newly completed node
	nodeMap := make(map[uuid.UUID]r_model.WorkflowNode)
	for _, node := range allNodes {
		if node.ID == completedNodeID {
			node.State = r_model.WorkflowNodeStateCompleted
		}
		nodeMap[node.ID] = node
	}

	var newReadyNodes []r_model.WorkflowNode
	var unlockedNodes []r_model.WorkflowNode

	// Check each locked node to see if its dependencies are now met
	for _, node := range allNodes {
		if node.State != r_model.WorkflowNodeStateLocked {
			continue
		}

		if sm.areDependenciesMet(node.DependsOn, nodeMap) {
			node.State = r_model.WorkflowNodeStateReady
			newReadyNodes = append(newReadyNodes, node)
			unlockedNodes = append(unlockedNodes, node)
		}
	}

	return newReadyNodes, unlockedNodes
}

// areDependenciesMet checks if all dependencies for a node are in COMPLETED state.
func (sm *WorkflowNodeStateMachine) areDependenciesMet(
	dependsOn []uuid.UUID,
	nodeMap map[uuid.UUID]r_model.WorkflowNode,
) bool {
	for _, depID := range dependsOn {
		depNode, exists := nodeMap[depID]
		if !exists {
			return false
		}
		if depNode.State != r_model.WorkflowNodeStateCompleted {
			return false
		}
	}
	return true
}

// areAllNodesCompleted checks if all nodes are in COMPLETED state, considering pending updates.
func (sm *WorkflowNodeStateMachine) areAllNodesCompleted(
	allNodes []r_model.WorkflowNode,
	updatedNodes []r_model.WorkflowNode,
) bool {
	// Build map of updated states
	updatedStateMap := make(map[uuid.UUID]r_model.WorkflowNodeState)
	for _, node := range updatedNodes {
		updatedStateMap[node.ID] = node.State
	}

	// Check all nodes
	for _, node := range allNodes {
		state := node.State
		if updatedState, wasUpdated := updatedStateMap[node.ID]; wasUpdated {
			state = updatedState
		}
		if state != r_model.WorkflowNodeStateCompleted {
			return false
		}
	}

	return true
}

// canTransitionToCompleted checks if a node can transition to COMPLETED from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToCompleted(currentState r_model.WorkflowNodeState) bool {
	// Only READY or IN_PROGRESS nodes can be completed
	return currentState == r_model.WorkflowNodeStateReady ||
		currentState == r_model.WorkflowNodeStateInProgress
}

// canTransitionToFailed checks if a node can transition to FAILED from its current state.
func (sm *WorkflowNodeStateMachine) canTransitionToFailed(currentState r_model.WorkflowNodeState) bool {
	// Any non-terminal state can transition to FAILED
	return currentState != r_model.WorkflowNodeStateFailed &&
		currentState != r_model.WorkflowNodeStateCompleted
}

// sortNodesByID sorts workflow nodes by ID to ensure consistent ordering and prevent deadlocks.
func (sm *WorkflowNodeStateMachine) sortNodesByID(nodes []r_model.WorkflowNode) {
	n := len(nodes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if nodes[j].ID.String() > nodes[j+1].ID.String() {
				nodes[j], nodes[j+1] = nodes[j+1], nodes[j]
			}
		}
	}
}
