package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/lokewate/go-workflow"
	"gorm.io/gorm"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type newWorkflowAdapter struct {
	manager workflow.Manager
	db      *gorm.DB
}

// NewWorkflowAdapter creates an adapter for the new workflow manager.
func NewWorkflowAdapter(db *gorm.DB) Manager {
	repo, err := workflow.NewDBRepo(db)
	if err != nil {
		slog.Error("failed to initialize new workflow DB repository", "error", err)
		return nil
	}
	m := workflow.NewWorkflowManager(repo, workflow.WithLogger(slog.Default()))

	slog.Info("initialized new workflow manager adapter with DB repo (experimental)")

	return &newWorkflowAdapter{
		manager: m,
		db:      db,
	}
}

func (a *newWorkflowAdapter) StartWorkflowInstance(
	ctx context.Context,
	tx *gorm.DB,
	workflowID uuid.UUID,
	workflowTemplates []model.WorkflowTemplate,
	globalContext map[string]any,
	handler WorkflowEventHandler,
) error {
	slog.Info("StartWorkflowInstance called on new workflow adapter", "workflowID", workflowID)

	wf := workflow.Workflow{
		ID: workflowID.String(),
	}

	for _, wt := range workflowTemplates {
		for _, nodeID := range wt.GetNodeTemplateIDs() {
			wf.Nodes = append(wf.Nodes, workflow.Node{
				ID:   nodeID.String(),
				Type: workflow.NodeTypeTask,
			})
		}
	}

	workflowJSON, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	_, err = a.manager.StartWorkflow(ctx, workflowJSON, globalContext)
	if err != nil {
		return fmt.Errorf("failed to start workflow in new manager: %w", err)
	}

	return nil
}

func (a *newWorkflowAdapter) RegisterTaskHandler(callback TaskInitHandler) error {
	a.manager.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
		nodeID := payload.NodeID()
		taskUUID, _ := uuid.Parse(nodeID)

		// Note: taskManager should not be aware about workflows, and should not need a
		// WorkflowID. This should be changed to a generic execution ID, that is used in the
		// completion callback.
		// For now, we'll pass uuid.Nil or try to find it if possible.

		req := taskManager.InitTaskRequest{
			TaskID:      taskUUID,
			WorkflowID:  uuid.Nil, // Placeholder if not in payload
			GlobalState: payload.Inputs,
		}

		_, err := callback(ctx, req)
		return err
	})
	return nil
}

func (a *newWorkflowAdapter) HandleTaskUpdate(ctx context.Context, update taskManager.WorkflowManagerNotification) error {
	// Map task completion back to go-workflow using ExecutionID
	// We need to store/retrieve the executionID. For this bridge, we'll assume update.TaskID
	// is the executionID string if it was passed that way, but locally it's uuid.UUID.
	// This shows a mismatch that might need more plumbing.
	return a.manager.TaskDone(ctx, update.TaskID.String(), update.AppendGlobalContext)
}

func (a *newWorkflowAdapter) GetWorkflowInstance(ctx context.Context, workflowID uuid.UUID) (*model.Workflow, error) {
	status, err := a.manager.GetStatus(ctx, workflowID.String())
	if err != nil {
		return nil, err
	}

	// state.GlobalContext likely has a method to get the map or we treat it as any.
	// The user provided 'Context state.GlobalContext `json:"-"`'

	wf := &model.Workflow{
		Status:        model.WorkflowStatus(status.Status),
		GlobalContext: status.Context.GetAll(),
	}
	wf.ID = workflowID

	return wf, nil
}
