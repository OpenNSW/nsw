package manager

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// NodeTemplateProvider defines the minimal interface for retrieving node template metadata.
type NodeTemplateProvider interface {
	GetWorkflowNodeTemplateByID(ctx context.Context, id uuid.UUID) (*model.WorkflowNodeTemplate, error)
	GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []uuid.UUID) ([]model.WorkflowNodeTemplate, error)
	GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error)
}

// WorkflowNodeRepository defines the workflow node data access methods used by the manager.
type WorkflowNodeRepository interface {
	GetWorkflowNodeByIDInTx(ctx context.Context, tx *gorm.DB, nodeID uuid.UUID) (*model.WorkflowNode, error)
	GetWorkflowNodesByIDsInTx(ctx context.Context, tx *gorm.DB, nodeIDs []uuid.UUID) ([]model.WorkflowNode, error)
	CreateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) ([]model.WorkflowNode, error)
	UpdateWorkflowNodesInTx(ctx context.Context, tx *gorm.DB, nodes []model.WorkflowNode) error
	GetWorkflowNodesByWorkflowIDInTx(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID) ([]model.WorkflowNode, error)
	GetWorkflowNodesByWorkflowIDsInTx(ctx context.Context, tx *gorm.DB, workflowIDs []uuid.UUID) ([]model.WorkflowNode, error)
	CountIncompleteNodesByWorkflowID(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID) (int64, error)
}

// WorkflowEventHandler defines the domain callbacks the generic manager invokes.
type WorkflowEventHandler interface {
	OnWorkflowStatusChanged(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID, fromStatus model.WorkflowStatus, toStatus model.WorkflowStatus, workflow *model.Workflow) error
}

// TaskInitHandler registers READY workflow nodes with the task manager.
type TaskInitHandler func(ctx context.Context, request taskManager.InitTaskRequest) (*taskManager.InitTaskResponse, error)

// Manager defines the public contract for the generic workflow engine.
type Manager interface {
	StartWorkflowInstance(ctx context.Context, tx *gorm.DB, workflowID uuid.UUID, workflowTemplates []model.WorkflowTemplate, goWorkflowTemplate *model.GoWorkflowTemplate, globalContext map[string]any, handler WorkflowEventHandler) error
	RegisterTaskHandler(callback TaskInitHandler) error
	HandleTaskUpdate(ctx context.Context, update taskManager.WorkflowManagerNotification) error
	GetWorkflowInstance(ctx context.Context, workflowID uuid.UUID) (*model.Workflow, error)
}

// workflowManager is the generic workflow engine implementation.
// It has no knowledge of domain concepts like consignments or pre-consignments.
type workflowManager struct {
	stateMachine         *WorkflowNodeStateMachine
	nodeRepo             WorkflowNodeRepository
	nodeTemplateProvider NodeTemplateProvider
	handlerMap           map[uuid.UUID]WorkflowEventHandler
	initTaskCallback     TaskInitHandler
	mu                   sync.RWMutex
	db                   *gorm.DB
}

var _ Manager = (*workflowManager)(nil)

// NewManager creates a new generic workflow manager.
func NewManager(
	db *gorm.DB,
	nodeRepo WorkflowNodeRepository,
	nodeTemplateProvider NodeTemplateProvider,
) Manager {
	m := &workflowManager{
		stateMachine:         NewWorkflowNodeStateMachine(nodeRepo),
		nodeRepo:             nodeRepo,
		nodeTemplateProvider: nodeTemplateProvider,
		handlerMap:           make(map[uuid.UUID]WorkflowEventHandler),
		db:                   db,
	}
	return m
}

// RegisterTaskHandler registers the callback used to initialize tasks for READY workflow nodes.
func (m *workflowManager) RegisterTaskHandler(callback TaskInitHandler) error {
	if callback == nil {
		return fmt.Errorf("init task callback cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.initTaskCallback = callback
	return nil
}

// StartWorkflowInstance initializes a workflow instance for a given consignment.
// It supports both traditional workflow templates and new go-workflow definitions.
func (m *workflowManager) StartWorkflowInstance(
	ctx context.Context,
	tx *gorm.DB,
	workflowID uuid.UUID,
	workflowTemplates []model.WorkflowTemplate,
	goWorkflowTemplate *model.GoWorkflowTemplate,
	globalContext map[string]any,
	handler WorkflowEventHandler,
) error {
	if handler == nil {
		return fmt.Errorf("workflow event handler cannot be nil")
	}

	if globalContext == nil {
		globalContext = make(map[string]any)
	}

	wf := &model.Workflow{
		Status:        model.WorkflowStatusInProgress,
		GlobalContext: globalContext,
	}
	wf.ID = workflowID
	if err := tx.Create(wf).Error; err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	uniqueNodeTemplateIDs := make(map[uuid.UUID]bool)
	var depEndNodeTemplateIDs model.UUIDArray
	for _, wt := range workflowTemplates {
		for _, nodeTemplateID := range wt.GetNodeTemplateIDs() {
			uniqueNodeTemplateIDs[nodeTemplateID] = true
		}
		if wt.EndNodeTemplateID != nil {
			depEndNodeTemplateIDs = append(depEndNodeTemplateIDs, *wt.EndNodeTemplateID)
		}
	}

	nodeTemplateIDsList := make([]uuid.UUID, 0, len(uniqueNodeTemplateIDs))
	for id := range uniqueNodeTemplateIDs {
		nodeTemplateIDsList = append(nodeTemplateIDsList, id)
	}

	nodeTemplates, err := m.nodeTemplateProvider.GetWorkflowNodeTemplatesByIDs(ctx, nodeTemplateIDsList)
	if err != nil {
		return fmt.Errorf("failed to retrieve workflow node templates: %w", err)
	}

	if len(depEndNodeTemplateIDs) > 0 {
		endNodeTemplate, err := m.nodeTemplateProvider.GetEndNodeTemplate(ctx)
		if err != nil {
			return fmt.Errorf("failed to get end node template: %w", err)
		}
		endNodeTemplate.DependsOn = depEndNodeTemplateIDs
		nodeTemplates = append(nodeTemplates, *endNodeTemplate)
	}

	_, newReadyNodes, endNodeID, err := m.stateMachine.InitializeNodesFromTemplates(ctx, tx, workflowID, nodeTemplates)
	if err != nil {
		return fmt.Errorf("failed to initialize workflow nodes: %w", err)
	}

	if endNodeID != nil {
		wf.EndNodeID = endNodeID
		if err := tx.Save(wf).Error; err != nil {
			return fmt.Errorf("failed to update workflow end node: %w", err)
		}
	}

	if len(newReadyNodes) > 0 {
		if err := m.registerNodesWithTaskManager(ctx, newReadyNodes, globalContext); err != nil {
			return fmt.Errorf("failed to register workflow nodes with task manager: %w", err)
		}
	}

	m.mu.Lock()
	m.handlerMap[workflowID] = handler
	m.mu.Unlock()

	if err := handler.OnWorkflowStatusChanged(ctx, tx, workflowID, "", model.WorkflowStatusInProgress, wf); err != nil {
		return fmt.Errorf("event handler OnWorkflowStatusChanged failed for workflow start: %w", err)
	}

	return nil
}

// GetWorkflowInstance returns the Workflow with preloaded WorkflowNodes and their templates.
func (m *workflowManager) GetWorkflowInstance(ctx context.Context, workflowID uuid.UUID) (*model.Workflow, error) {
	var wf model.Workflow
	if err := m.db.WithContext(ctx).
		Preload("WorkflowNodes.WorkflowNodeTemplate").
		First(&wf, "id = ?", workflowID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow %s: %w", workflowID, err)
	}
	return &wf, nil
}

// HandleTaskUpdate processes a single task notification sent by the task manager callback.
func (m *workflowManager) HandleTaskUpdate(ctx context.Context, update taskManager.WorkflowManagerNotification) error {
	if update.UpdatedState == nil {
		return fmt.Errorf("received nil state in workflow node update for task %s", update.TaskID)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	workflowState, err := pluginStateToWorkflowNodeState(*update.UpdatedState)
	if err != nil {
		return fmt.Errorf("invalid state in workflow node update for task %s: %w", update.TaskID, err)
	}

	taskUUID, err := uuid.Parse(update.TaskID)
	if err != nil {
		return fmt.Errorf("invalid task ID format for workflow manager: %w", err)
	}

	updateReq := model.UpdateWorkflowNodeDTO{
		WorkflowNodeID:      taskUUID,
		State:               workflowState,
		AppendGlobalContext: update.AppendGlobalContext,
		ExtendedState:       update.ExtendedState,
		Outcome:             update.Outcome,
	}

	node, err := m.nodeRepo.GetWorkflowNodeByIDInTx(ctx, m.db, taskUUID)
	if err != nil {
		return fmt.Errorf("failed to look up workflow node for task %s: %w", update.TaskID, err)
	}

	handler := m.findHandler(node.WorkflowID)
	if handler == nil {
		return fmt.Errorf("no event handler found for workflow %s task %s", node.WorkflowID, update.TaskID)
	}

	newReadyNodes, globalContext, err := m.processStateTransition(ctx, node, &updateReq, handler)
	if err != nil {
		return fmt.Errorf("failed to process workflow node state transition for task %s state %s: %w", update.TaskID, workflowState, err)
	}

	if len(newReadyNodes) > 0 {
		if err := m.registerNodesWithTaskManager(ctx, newReadyNodes, globalContext); err != nil {
			return fmt.Errorf("failed to register new ready nodes with task manager for task %s: %w", update.TaskID, err)
		}
	}

	return nil
}

func (m *workflowManager) processStateTransition(
	ctx context.Context,
	node *model.WorkflowNode,
	updateReq *model.UpdateWorkflowNodeDTO,
	handler WorkflowEventHandler,
) ([]model.WorkflowNode, map[string]any, error) {
	tx := m.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var wf model.Workflow
	if err := tx.First(&wf, "id = ?", node.WorkflowID).Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to load workflow %s: %w", node.WorkflowID, err)
	}

	if len(updateReq.AppendGlobalContext) > 0 {
		if wf.GlobalContext == nil {
			wf.GlobalContext = make(map[string]any)
		}
		maps.Copy(wf.GlobalContext, updateReq.AppendGlobalContext)
		if err := tx.Save(&wf).Error; err != nil {
			tx.Rollback()
			return nil, nil, fmt.Errorf("failed to update workflow global context: %w", err)
		}
	}

	workflowNode, err := m.nodeRepo.GetWorkflowNodeByIDInTx(ctx, tx, updateReq.WorkflowNodeID)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to retrieve workflow node %s: %w", updateReq.WorkflowNodeID, err)
	}

	var newReadyNodes []model.WorkflowNode

	switch updateReq.State {
	case model.WorkflowNodeStateFailed:
		if workflowNode.State != model.WorkflowNodeStateFailed {
			if err := m.stateMachine.TransitionToFailed(ctx, tx, workflowNode, updateReq); err != nil {
				tx.Rollback()
				return nil, nil, fmt.Errorf("failed to transition node to FAILED: %w", err)
			}
			fromStatus := wf.Status
			wf.Status = model.WorkflowStatusFailed
			if err := tx.Save(&wf).Error; err != nil {
				tx.Rollback()
				return nil, nil, fmt.Errorf("failed to mark workflow as failed: %w", err)
			}
			if err := handler.OnWorkflowStatusChanged(ctx, tx, node.WorkflowID, fromStatus, model.WorkflowStatusFailed, &wf); err != nil {
				tx.Rollback()
				return nil, nil, fmt.Errorf("event handler OnWorkflowStatusChanged failed for workflow failure: %w", err)
			}
		}

	case model.WorkflowNodeStateInProgress:
		if err := m.stateMachine.TransitionToInProgress(ctx, tx, workflowNode, updateReq); err != nil {
			tx.Rollback()
			return nil, nil, fmt.Errorf("failed to transition node to IN_PROGRESS: %w", err)
		}

	case model.WorkflowNodeStateCompleted:
		if workflowNode.State != model.WorkflowNodeStateCompleted {
			completionConfig := WorkflowCompletionConfig{EndNodeID: wf.EndNodeID}
			result, err := m.stateMachine.TransitionToCompleted(ctx, tx, workflowNode, updateReq, &completionConfig)
			if err != nil {
				tx.Rollback()
				return nil, nil, fmt.Errorf("failed to transition node to COMPLETED: %w", err)
			}
			newReadyNodes = result.NewReadyNodes

			if result.WorkflowFinished {
				fromStatus := wf.Status
				wf.Status = model.WorkflowStatusCompleted
				if err := tx.Save(&wf).Error; err != nil {
					tx.Rollback()
					return nil, nil, fmt.Errorf("failed to mark workflow as completed: %w", err)
				}
				if err := handler.OnWorkflowStatusChanged(ctx, tx, node.WorkflowID, fromStatus, model.WorkflowStatusCompleted, &wf); err != nil {
					tx.Rollback()
					return nil, nil, fmt.Errorf("event handler OnWorkflowStatusChanged failed for workflow completion: %w", err)
				}
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newReadyNodes, wf.GlobalContext, nil
}

func (m *workflowManager) findHandler(workflowID uuid.UUID) WorkflowEventHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handlerMap[workflowID]
}

func (m *workflowManager) registerNodesWithTaskManager(ctx context.Context, workflowNodes []model.WorkflowNode, globalContext map[string]any) error {
	m.mu.RLock()
	initTaskCallback := m.initTaskCallback
	m.mu.RUnlock()

	if initTaskCallback == nil {
		return fmt.Errorf("task manager callback is not configured")
	}

	for _, node := range workflowNodes {
		nodeTemplate, err := m.nodeTemplateProvider.GetWorkflowNodeTemplateByID(ctx, node.WorkflowNodeTemplateID)
		if err != nil {
			return fmt.Errorf("failed to get workflow node template %s: %w", node.WorkflowNodeTemplateID, err)
		}
		initTaskRequest := taskManager.InitTaskRequest{
			TaskID:                 node.ID.String(),
			WorkflowID:             node.WorkflowID,
			WorkflowNodeTemplateID: node.WorkflowNodeTemplateID,
			Type:                   nodeTemplate.Type,
			GlobalState:            globalContext,
			Config:                 nodeTemplate.Config,
		}
		response, err := initTaskCallback(ctx, initTaskRequest)
		if err != nil {
			return fmt.Errorf("failed to initialize task for node %s: %w", node.ID, err)
		}
		slog.Info("registered workflow node with task manager", "nodeID", node.ID, "response", response.Result)
	}
	return nil
}

func pluginStateToWorkflowNodeState(state plugin.State) (model.WorkflowNodeState, error) {
	switch state {
	case plugin.Initialized:
		return model.WorkflowNodeStateReady, nil
	case plugin.InProgress:
		return model.WorkflowNodeStateInProgress, nil
	case plugin.Completed:
		return model.WorkflowNodeStateCompleted, nil
	case plugin.Failed:
		return model.WorkflowNodeStateFailed, nil
	default:
		return "", fmt.Errorf("unknown plugin state: %s", state)
	}
}
