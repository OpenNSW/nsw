// Tests for bootstrap.Runtime.
//
// NewRuntime invokes engine.NewTemporalManager and orchestrator.NewTaskManager
// directly (no factory hooks), and StartWorker performs a live gRPC handshake
// against a Temporal cluster. As a result the happy-path of NewRuntime, the
// parentCompletion → UpstreamService wiring, and the closure handlers
// (parentTaskHandler / taskHandler) cannot be exercised in a pure unit test —
// covering them needs an in-memory Temporal harness and belongs alongside the
// other Temporal integration suites.
//
// This file covers everything that CAN be tested without that infrastructure:
//
//   - Config input validation (the three early returns at the top of NewRuntime).
//   - Runtime.Close behaviour: nil-receiver safety, nil-manager safety, and
//     StopWorker delegation when both managers are wired.
//   - The three accessors (Manager / ParentManager / Registry) returning the
//     pointers they were constructed with.
//   - The UpstreamService interface contract.
package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"

	engine "github.com/OpenNSW/go-temporal-workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"
)

// ─── test doubles ──────────────────────────────────────────────────────────

// fakeTemporalClient satisfies client.Client via interface embedding. Any
// real method call would panic on the nil embedded interface, which is fine —
// our tests only need a non-nil reference so NewRuntime's nil-check passes.
type fakeTemporalClient struct{ client.Client }

// fakeTaskStore satisfies tfstore.TaskStore the same way.
type fakeTaskStore struct{ tfstore.TaskStore }

// fakeTemporalManager records StartWorker / StopWorker invocations so the
// Close test can assert both managers were stopped exactly once. The remaining
// TemporalManager methods are stubbed to satisfy the interface; tests that
// touch them would belong in an integration suite.
type fakeTemporalManager struct {
	mu         sync.Mutex
	startCalls int
	stopCalls  int
	startErr   error
}

func (m *fakeTemporalManager) StartWorker() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls++
	return m.startErr
}

func (m *fakeTemporalManager) StopWorker() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalls++
}

func (m *fakeTemporalManager) StartWorkflow(_ context.Context, _ string, _ engine.WorkflowDefinition, _ map[string]any) error {
	return nil
}

func (m *fakeTemporalManager) TaskDone(_ context.Context, _, _, _ string, _ map[string]any) error {
	return nil
}

func (m *fakeTemporalManager) TaskUpdate(_ context.Context, _, _ string, _ engine.UpdateEvent) error {
	return nil
}

func (m *fakeTemporalManager) GetStatus(_ context.Context, _ string) (*engine.WorkflowInstance, error) {
	return nil, nil
}

func (m *fakeTemporalManager) stopCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopCalls
}

// recordingUpstream captures CompletionHandler invocations and the result
// returned to the caller. Documents the UpstreamService contract; an
// integration test can wire it through a real workflow to verify the parent
// completion handler delegates correctly.
type recordingUpstream struct {
	mu    sync.Mutex
	calls []upstreamCall
	err   error
}

type upstreamCall struct {
	workflowID string
	finalCtx   map[string]any
}

func (u *recordingUpstream) CompletionHandler(workflowID string, finalCtx map[string]any) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.calls = append(u.calls, upstreamCall{workflowID: workflowID, finalCtx: finalCtx})
	return u.err
}

// ─── NewRuntime input validation ───────────────────────────────────────────

func TestNewRuntime_RequiresTemporalClient(t *testing.T) {
	_, err := NewRuntime(Config{
		TemporalClient: nil,
		Store:          &fakeTaskStore{},
		Registry:       orchestrator.NewTaskTemplateRegistry(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "temporal client is required")
}

func TestNewRuntime_RequiresStore(t *testing.T) {
	_, err := NewRuntime(Config{
		TemporalClient: &fakeTemporalClient{},
		Store:          nil,
		Registry:       orchestrator.NewTaskTemplateRegistry(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task store is required")
}

func TestNewRuntime_RequiresRegistry(t *testing.T) {
	_, err := NewRuntime(Config{
		TemporalClient: &fakeTemporalClient{},
		Store:          &fakeTaskStore{},
		Registry:       nil,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template registry is required")
}

// All three validation errors are surfaced *before* any side effect — they
// must not start a worker or touch the store. A loose smoke check that the
// returned runtime pointer is nil reinforces the "no partial construction"
// guarantee.
func TestNewRuntime_ValidationFailureLeavesNoPartialRuntime(t *testing.T) {
	r, err := NewRuntime(Config{}) // every required field nil
	require.Error(t, err)
	assert.Nil(t, r)
}

// ─── Runtime.Close ─────────────────────────────────────────────────────────

func TestRuntime_Close_NilReceiver_IsNoOp(t *testing.T) {
	var r *Runtime
	assert.NoError(t, r.Close(), "Close on a nil *Runtime must not panic and must return nil")
}

func TestRuntime_Close_NilManagers_IsNoOp(t *testing.T) {
	r := &Runtime{}
	assert.NoError(t, r.Close())
}

func TestRuntime_Close_StopsBothManagers(t *testing.T) {
	parent := &fakeTemporalManager{}
	task := &fakeTemporalManager{}

	r := &Runtime{parent: parent, task: task}

	require.NoError(t, r.Close())
	assert.Equal(t, 1, parent.stopCallCount(), "parent worker should be stopped exactly once")
	assert.Equal(t, 1, task.stopCallCount(), "task worker should be stopped exactly once")
}

// Close should stop the task worker even when the parent manager is nil and
// vice-versa — the order in the source is task first, then parent, so a
// partially-wired runtime should still tear down what it has.
func TestRuntime_Close_OnlyTaskWired(t *testing.T) {
	task := &fakeTemporalManager{}
	r := &Runtime{task: task}

	require.NoError(t, r.Close())
	assert.Equal(t, 1, task.stopCallCount())
}

func TestRuntime_Close_OnlyParentWired(t *testing.T) {
	parent := &fakeTemporalManager{}
	r := &Runtime{parent: parent}

	require.NoError(t, r.Close())
	assert.Equal(t, 1, parent.stopCallCount())
}

// Calling Close twice should not double-stop in a way that breaks anything —
// the second invocation just stops the workers again. We assert the call
// counter to document the current (non-idempotent) behaviour so a future
// idempotency change shows up here.
func TestRuntime_Close_IsRepeatable(t *testing.T) {
	parent := &fakeTemporalManager{}
	task := &fakeTemporalManager{}
	r := &Runtime{parent: parent, task: task}

	require.NoError(t, r.Close())
	require.NoError(t, r.Close())

	assert.Equal(t, 2, parent.stopCallCount())
	assert.Equal(t, 2, task.stopCallCount())
}

// ─── Accessors ─────────────────────────────────────────────────────────────

func TestRuntime_Accessors_ReturnWiredValues(t *testing.T) {
	parent := &fakeTemporalManager{}
	task := &fakeTemporalManager{}
	registry := orchestrator.NewTaskTemplateRegistry()

	// TaskManager is concrete with required deps — Manager() simply returns
	// whatever pointer it was constructed with, so nil is sufficient to
	// verify the accessor itself.
	r := &Runtime{
		tm:       nil,
		parent:   parent,
		task:     task,
		registry: registry,
	}

	assert.Nil(t, r.TaskManager(), "Manager() returns the underlying *orchestrator.TaskManager pointer")
	assert.Same(t, engine.TemporalManager(parent), r.WorkflowManager(), "ParentManager() returns the parent manager passed at construction")
	assert.Same(t, registry, r.Registry(), "Registry() returns the template registry passed at construction")
}

// ─── UpstreamService contract ──────────────────────────────────────────────

// The UpstreamService interface is invoked only from parentCompletion (a
// closure inside NewRuntime), so we can't drive it end-to-end without a live
// Temporal. We can still pin the interface shape so that any signature change
// fails compilation here.
func TestUpstreamService_ContractCompiles(t *testing.T) {
	var svc UpstreamService = &recordingUpstream{}

	err := svc.CompletionHandler("wf-1", map[string]any{"npqs.payment_status": "success"})
	require.NoError(t, err)

	rec := svc.(*recordingUpstream)
	require.Len(t, rec.calls, 1)
	assert.Equal(t, "wf-1", rec.calls[0].workflowID)
	assert.Equal(t, map[string]any{"npqs.payment_status": "success"}, rec.calls[0].finalCtx)
}

// When the upstream returns an error, the implementer must surface it
// unchanged — parentCompletion in runtime.go wraps it with fmt.Errorf("%w"),
// so the caller's errors.Is should still match.
func TestUpstreamService_ErrorPassthrough(t *testing.T) {
	sentinel := errors.New("upstream rejected")
	svc := &recordingUpstream{err: sentinel}

	err := svc.CompletionHandler("wf-2", nil)
	assert.ErrorIs(t, err, sentinel)
}
