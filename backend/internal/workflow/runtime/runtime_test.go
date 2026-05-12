package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	workflowmanager "github.com/OpenNSW/go-temporal-workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type fakeTemporalManager struct {
	startErr       error
	startCalled    bool
	stopCalled     bool
	taskDoneCalled bool
	taskDoneErr    error
	taskDoneInput  struct {
		workflowID string
		taskID     string
		outputs    map[string]any
	}
}

func (m *fakeTemporalManager) StartWorkflow(_ context.Context, _ string, _ workflowmanager.WorkflowDefinition, _ map[string]any) error {
	return nil
}

func (m *fakeTemporalManager) TaskDone(_ context.Context, workflowID, _ string, nodeID string, output map[string]any) error {
	m.taskDoneCalled = true
	m.taskDoneInput.workflowID = workflowID
	m.taskDoneInput.taskID = nodeID
	m.taskDoneInput.outputs = output
	return m.taskDoneErr
}

func (m *fakeTemporalManager) TaskUpdate(_ context.Context, _ string, _ string, _ workflowmanager.UpdateEvent) error {
	return nil
}

func (m *fakeTemporalManager) GetStatus(_ context.Context, _ string) (*workflowmanager.WorkflowInstance, error) {
	return nil, nil
}

func (m *fakeTemporalManager) StartWorker() error {
	m.startCalled = true
	return m.startErr
}

func (m *fakeTemporalManager) StopWorker() {
	m.stopCalled = true
}

type fakeTemplateProvider struct {
	template *model.WorkflowNodeTemplate
	err      error
	lastCtx  context.Context
	lastID   string
}

func (p *fakeTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(_ context.Context, _ string, _ model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	return nil, nil
}

func (p *fakeTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlowV2(_ context.Context, _ string, _ model.ConsignmentFlow) (*model.WorkflowTemplateV2, error) {
	return nil, nil
}

func (p *fakeTemplateProvider) GetWorkflowTemplateByID(_ context.Context, _ string) (*model.WorkflowTemplate, error) {
	return nil, nil
}

func (p *fakeTemplateProvider) GetWorkflowTemplateByIDV2(_ context.Context, _ string) (*model.WorkflowTemplateV2, error) {
	return nil, nil
}

func (p *fakeTemplateProvider) GetWorkflowNodeTemplatesByIDs(_ context.Context, _ []string) ([]model.WorkflowNodeTemplate, error) {
	return nil, nil
}

func (p *fakeTemplateProvider) GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error) {
	p.lastCtx = ctx
	p.lastID = id
	if p.err != nil {
		return nil, p.err
	}
	return p.template, nil
}

func (p *fakeTemplateProvider) GetEndNodeTemplate(_ context.Context) (*model.WorkflowNodeTemplate, error) {
	return nil, nil
}

type fakeTaskManager struct {
	doneCallback taskManager.WorkflowDoneHandler
	initErr      error
	lastInitCtx  context.Context
	lastInitReq  taskManager.InitTaskRequest
	initCtxErr   error
}

type fakeUpstreamService struct {
	completionCalled bool
	workflowID       string
	finalContext     map[string]any
	err              error
}

func (s *fakeUpstreamService) CompletionHandler(workflowID string, finalContext map[string]any) error {
	s.completionCalled = true
	s.workflowID = workflowID
	s.finalContext = finalContext
	return s.err
}

func (m *fakeTaskManager) InitTask(ctx context.Context, request taskManager.InitTaskRequest) (*taskManager.InitTaskResponse, error) {
	m.lastInitCtx = ctx
	m.lastInitReq = request
	m.initCtxErr = ctx.Err()
	if m.initErr != nil {
		return nil, m.initErr
	}
	return &taskManager.InitTaskResponse{Success: true}, nil
}

func (m *fakeTaskManager) ExecuteTask(_ context.Context, _ taskManager.ExecuteTaskRequest) (*plugin.ExecutionResponse, error) {
	return nil, nil
}

func (m *fakeTaskManager) GetTaskRenderInfo(_ context.Context, _ string) (*plugin.ApiResponse, error) {
	return nil, nil
}

func (m *fakeTaskManager) RegisterUpstreamDoneCallback(callback taskManager.WorkflowDoneHandler) {
	m.doneCallback = callback
}

func (m *fakeTaskManager) RegisterUpstreamUpdateCallback(_ taskManager.WorkflowUpdateHandler) {}

func TestNewRuntime_StartWorkerFailureReturnsError(t *testing.T) {
	fakeManager := &fakeTemporalManager{startErr: errors.New("start failed")}
	taskMgr := &fakeTaskManager{}
	templateProvider := &fakeTemplateProvider{template: &model.WorkflowNodeTemplate{}}

	_, err := newRuntimeWithFactory(taskMgr, templateProvider, func(
		_ workflowmanager.TaskActivationHandler,
		_ workflowmanager.WorkflowCompletionHandler,
	) workflowmanager.TemporalManager {
		return fakeManager
	}, nil)

	require.Error(t, err)
	assert.True(t, fakeManager.startCalled)
}

func TestRuntimeClose_StopsWorkerAndCancelsRuntimeContext(t *testing.T) {
	fakeManager := &fakeTemporalManager{}
	ctx, cancel := context.WithCancel(context.Background())

	r := &Runtime{
		manager:       fakeManager,
		runtimeCancel: cancel,
	}

	err := r.Close()
	require.NoError(t, err)
	assert.True(t, fakeManager.stopCalled)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}

func TestNewRuntime_ActivationHandlerInitializesTask(t *testing.T) {
	fakeManager := &fakeTemporalManager{}
	taskMgr := &fakeTaskManager{}
	templateProvider := &fakeTemplateProvider{template: &model.WorkflowNodeTemplate{
		BaseModel: model.BaseModel{ID: "template-1"},
		Type:      plugin.Type("test"),
		Config:    json.RawMessage(`{"x":1}`),
	}}

	var activationHandler func(payload workflowmanager.TaskPayload) error
	runtime, err := newRuntimeWithFactory(taskMgr, templateProvider, func(
		activation workflowmanager.TaskActivationHandler,
		_ workflowmanager.WorkflowCompletionHandler,
	) workflowmanager.TemporalManager {
		activationHandler = activation
		return fakeManager
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	require.NotNil(t, activationHandler)
	payload := workflowmanager.TaskPayload{
		NodeID:         "node-1",
		WorkflowID:     "wf-1",
		TaskTemplateID: "template-1",
		Inputs:         map[string]any{"a": "b"},
	}

	err = activationHandler(payload)
	require.NoError(t, err)
	assert.Equal(t, "template-1", templateProvider.lastID)
	require.NotNil(t, taskMgr.lastInitCtx)
	assert.NoError(t, taskMgr.initCtxErr)
	assert.Equal(t, payload.NodeID, taskMgr.lastInitReq.TaskID)
	assert.Equal(t, payload.WorkflowID, taskMgr.lastInitReq.WorkflowID)
	assert.Equal(t, "template-1", taskMgr.lastInitReq.WorkflowNodeTemplateID)
	assert.Equal(t, map[string]any{"a": "b"}, taskMgr.lastInitReq.GlobalState)
}

func TestNewRuntime_TaskDoneCallbackDelegatesToWorkflowManager(t *testing.T) {
	fakeManager := &fakeTemporalManager{}
	taskMgr := &fakeTaskManager{}
	templateProvider := &fakeTemplateProvider{template: &model.WorkflowNodeTemplate{}}

	runtime, err := newRuntimeWithFactory(taskMgr, templateProvider, func(
		_ workflowmanager.TaskActivationHandler,
		_ workflowmanager.WorkflowCompletionHandler,
	) workflowmanager.TemporalManager {
		return fakeManager
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	require.NotNil(t, taskMgr.doneCallback)
	taskMgr.doneCallback(context.Background(), "wf-1", "task-1", map[string]any{"ok": true})

	assert.True(t, fakeManager.taskDoneCalled)
	assert.Equal(t, "wf-1", fakeManager.taskDoneInput.workflowID)
	assert.Equal(t, "task-1", fakeManager.taskDoneInput.taskID)
	assert.Equal(t, map[string]any{"ok": true}, fakeManager.taskDoneInput.outputs)
}

func TestNewRuntime_CompletionHandlerDelegatesToUpstreamService(t *testing.T) {
	fakeManager := &fakeTemporalManager{}
	taskMgr := &fakeTaskManager{}
	templateProvider := &fakeTemplateProvider{template: &model.WorkflowNodeTemplate{}}
	upstreamService := &fakeUpstreamService{}

	var completionHandler workflowmanager.WorkflowCompletionHandler
	runtime, err := newRuntimeWithFactory(taskMgr, templateProvider, func(
		_ workflowmanager.TaskActivationHandler,
		completion workflowmanager.WorkflowCompletionHandler,
	) workflowmanager.TemporalManager {
		completionHandler = completion
		return fakeManager
	}, upstreamService)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close() })

	require.NotNil(t, completionHandler)
	err = completionHandler("wf-1", map[string]any{"status": "done"})
	require.NoError(t, err)
	assert.True(t, upstreamService.completionCalled)
	assert.Equal(t, "wf-1", upstreamService.workflowID)
	assert.Equal(t, map[string]any{"status": "done"}, upstreamService.finalContext)
}
