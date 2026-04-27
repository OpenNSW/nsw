package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	formmodel "github.com/OpenNSW/nsw/internal/form/model"
	"github.com/OpenNSW/nsw/pkg/remote"
)

// wfeAPI is a minimal API stub for WaitForEventTask tests.
type wfeAPI struct {
	pluginState     string
	taskID          string
	workflowID      string
	canTransition   func(action string) bool
	transitionCalls []string
	transitionErr   error
}

func (a *wfeAPI) GetTaskID() string                        { return a.taskID }
func (a *wfeAPI) GetWorkflowID() string                    { return a.workflowID }
func (a *wfeAPI) GetTaskState() State                      { return InProgress }
func (a *wfeAPI) GetPluginState() string                   { return a.pluginState }
func (a *wfeAPI) ReadFromGlobalStore(_ string) (any, bool) { return nil, false }
func (a *wfeAPI) WriteToLocalStore(_ string, _ any) error  { return nil }
func (a *wfeAPI) ReadFromLocalStore(_ string) (any, error) { return nil, nil }
func (a *wfeAPI) CanTransition(action string) bool {
	if a.canTransition != nil {
		return a.canTransition(action)
	}
	return true
}
func (a *wfeAPI) Transition(action string) error {
	a.transitionCalls = append(a.transitionCalls, action)
	return a.transitionErr
}

func (a *wfeAPI) calledWith(action string) bool {
	for _, c := range a.transitionCalls {
		if c == action {
			return true
		}
	}
	return false
}

type mockFormService struct {
	getFormByID func(ctx context.Context, formID string) (*formmodel.FormResponse, error)
}

func (m *mockFormService) GetFormByID(ctx context.Context, formID string) (*formmodel.FormResponse, error) {
	if m.getFormByID != nil {
		return m.getFormByID(ctx, formID)
	}
	return nil, nil
}

func newWFETask(t *testing.T, serverURL string) (*WaitForEventTask, *wfeAPI) {
	t.Helper()

	mgr := remote.NewManager()
	if serverURL != "" && serverURL != "http://irrelevant" {
		cfg := remote.Registry{
			Version: "1.0",
			Services: []remote.ServiceConfig{
				{
					ID:  "test-service",
					URL: serverURL,
				},
			},
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal registry: %v", err)
		}
		tmp := t.TempDir() + "/services.json"
		if err := os.WriteFile(tmp, data, 0644); err != nil {
			t.Fatalf("write services file: %v", err)
		}
		if err := mgr.LoadServices(tmp); err != nil {
			t.Fatalf("load services: %v", err)
		}
	}

	raw, err := json.Marshal(WaitForEventConfig{
		Submission: &SubmissionConfig{
			Url: serverURL,
			Request: &Request{
				TaskCode: "test-task-code",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	task, err := NewWaitForEventTask(raw, "http://localhost:8080", mgr, &mockFormService{})
	if err != nil {
		t.Fatalf("NewWaitForEventTask: %v", err)
	}
	api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString()}
	task.Init(api)
	return task, api
}

func TestWaitForEventTask_Start_NotifiesAndTransitions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(action string) bool { return action == FSMActionStart }

	resp, err := task.Start(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Message != "Notified external service, waiting for callback" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if !api.calledWith(FSMActionStart) {
		t.Error("expected Transition(FSMActionStart) to be called")
	}
	if api.calledWith(waitForEventFSMStartFailed) {
		t.Error("expected Transition(START_FAILED) NOT to be called")
	}
}

func TestWaitForEventTask_Start_NotificationFails_TransitionsToNotifyFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // non-retryable so the test doesn't wait on backoff
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(action string) bool { return action == FSMActionStart }

	resp, err := task.Start(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Message != "Failed to notify external service" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if !api.calledWith(waitForEventFSMStartFailed) {
		t.Error("expected Transition(START_FAILED) to be called")
	}
	if api.calledWith(FSMActionStart) {
		t.Error("expected Transition(FSMActionStart) NOT to be called")
	}
}

func TestWaitForEventTask_Start_AlreadyStarted(t *testing.T) {
	task, api := newWFETask(t, "http://irrelevant")
	api.canTransition = func(_ string) bool { return false }

	resp, err := task.Start(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Message != "WaitForEvent already started" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if len(api.transitionCalls) != 0 {
		t.Errorf("expected no Transition calls, got %v", api.transitionCalls)
	}
}

func TestWaitForEventTask_Execute_Complete(t *testing.T) {
	task, api := newWFETask(t, "http://irrelevant")

	resp, err := task.Execute(context.Background(), &ExecutionRequest{Action: waitForEventFSMComplete})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Message != "Task completed by external service" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if !api.calledWith(waitForEventFSMComplete) {
		t.Error("expected Transition(COMPLETE) to be called")
	}
}

func TestWaitForEventTask_Execute_Retry_Succeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)

	resp, err := task.Execute(context.Background(), &ExecutionRequest{Action: waitForEventFSMRetry})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Message != "Notified external service, waiting for callback" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if !api.calledWith(waitForEventFSMRetry) {
		t.Error("expected Transition(retry) to be called")
	}
}

func TestWaitForEventTask_Execute_Retry_NotificationFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // non-retryable so the test doesn't wait on backoff
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)

	resp, err := task.Execute(context.Background(), &ExecutionRequest{Action: waitForEventFSMRetry})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Failed to notify external service", resp.Message)
	require.NotNil(t, resp.ApiResponse)
	assert.False(t, resp.ApiResponse.Success)
	require.NotNil(t, resp.ApiResponse.Error)
	assert.Equal(t, "EXTERNAL_SERVICE_NOTIFICATION_FAILED", resp.ApiResponse.Error.Code)
	assert.Empty(t, api.transitionCalls, "no transition should occur when notification fails")
}

func TestWaitForEventTask_Execute_UnsupportedAction(t *testing.T) {
	task, _ := newWFETask(t, "http://irrelevant")

	resp, err := task.Execute(context.Background(), &ExecutionRequest{Action: "UNKNOWN"})

	require.ErrorContains(t, err, "unsupported action")
	assert.Nil(t, resp)
}

func TestWaitForEventTask_Execute_NilRequest(t *testing.T) {
	task, _ := newWFETask(t, "http://irrelevant")

	resp, err := task.Execute(context.Background(), nil)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}
}

// ── NewWaitForEventTask ───────────────────────────────────────────────────────

func TestNewWaitForEventTask_InvalidJSON(t *testing.T) {
	_, err := NewWaitForEventTask(json.RawMessage(`{invalid}`), "http://localhost:8080", nil, nil)
	require.Error(t, err)
}

// ── GetRenderInfo / renderContent ─────────────────────────────────────────────

func TestWaitForEventTask_GetRenderInfo_NoDisplay(t *testing.T) {
	task, api := newWFETask(t, "http://irrelevant")
	api.pluginState = string(notifiedService)

	resp, err := task.GetRenderInfo(context.Background())

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)

	data, ok := resp.Data.(GetRenderInfoResponse)
	require.True(t, ok)
	assert.Equal(t, TaskTypeWaitForEvent, data.Type)
	assert.Equal(t, string(notifiedService), data.PluginState)

	content, ok := data.Content.(map[string]any)
	require.True(t, ok)
	assert.Empty(t, content, "content should be empty map when display is not configured")
}

func TestWaitForEventTask_GetRenderInfo_WithDisplay(t *testing.T) {
	raw, err := json.Marshal(WaitForEventConfig{
		Submission: &SubmissionConfig{
			Url: "http://irrelevant",
		},
		Display: &WaitForEventDisplay{
			Title:       "Awaiting verification",
			Description: "Please wait while we process your request.",
		},
	})
	require.NoError(t, err)

	task, taskErr := NewWaitForEventTask(raw, "http://localhost:8080", nil, nil)
	require.NoError(t, taskErr)
	api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString(), pluginState: string(notifiedService)}
	task.Init(api)

	resp, err := task.GetRenderInfo(context.Background())

	require.NoError(t, err)
	content, ok := resp.Data.(GetRenderInfoResponse).Content.(map[string]any)
	require.True(t, ok)

	display, ok := content["display"].(*WaitForEventDisplay)
	require.True(t, ok, "display should be a *WaitForEventDisplay")
	assert.Equal(t, "Awaiting verification", display.Title)
	assert.Equal(t, "Please wait while we process your request.", display.Description)
}

func TestWaitForEventTask_GetRenderInfo_WithDynamicDisplay_UsesPluginState(t *testing.T) {
	tests := []struct {
		name          string
		pluginState   string
		expectedTitle string
		expectedDesc  string
	}{
		{
			name:          "waiting state",
			pluginState:   string(notifiedService),
			expectedTitle: "Waiting on Sample Drop Off Confirmation",
			expectedDesc:  "Please drop off your sample at the designated location and confirm the drop off by clicking on this task",
		},
		{
			name:          "failed state",
			pluginState:   string(notifyFailed),
			expectedTitle: "Sample Drop Off Confirmation Failed",
			expectedDesc:  "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.",
		},
		{
			name:          "completed state",
			pluginState:   string(receivedCallback),
			expectedTitle: "Sample Drop Off Confirmation Completed",
			expectedDesc:  "We have received confirmation of your sample drop off. We will notify you once the test results are available.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(WaitForEventConfig{
				Submission: &SubmissionConfig{Url: "http://irrelevant"},
				Display: &WaitForEventDisplay{
					Title: map[string]any{
						"waiting":   "Waiting on Sample Drop Off Confirmation",
						"failed":    "Sample Drop Off Confirmation Failed",
						"completed": "Sample Drop Off Confirmation Completed",
					},
					Description: map[string]any{
						"waiting":   "Please drop off your sample at the designated location and confirm the drop off by clicking on this task",
						"failed":    "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.",
						"completed": "We have received confirmation of your sample drop off. We will notify you once the test results are available.",
					},
				},
			})
			require.NoError(t, err)

			task, taskErr := NewWaitForEventTask(raw, "http://localhost:8080", nil, nil)
			require.NoError(t, taskErr)
			api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString(), pluginState: tt.pluginState}
			task.Init(api)

			resp, err := task.GetRenderInfo(context.Background())
			require.NoError(t, err)

			content, ok := resp.Data.(GetRenderInfoResponse).Content.(map[string]any)
			require.True(t, ok)
			display, ok := content["display"].(*WaitForEventDisplay)
			require.True(t, ok)
			assert.Equal(t, tt.expectedTitle, display.Title)
			assert.Equal(t, tt.expectedDesc, display.Description)
		})
	}
}

func TestWaitForEventTask_GetRenderInfo_WithMixedDisplayShape(t *testing.T) {
	raw, err := json.Marshal(WaitForEventConfig{
		Submission: &SubmissionConfig{Url: "http://irrelevant"},
		Display: &WaitForEventDisplay{
			Title: "Sample Drop Off Confirmation",
			Description: map[string]any{
				"waiting":   "Please drop off your sample at the designated location and confirm the drop off by clicking on this task",
				"failed":    "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.",
				"completed": "We have received confirmation of your sample drop off. We will notify you once the test results are available.",
			},
		},
	})
	require.NoError(t, err)

	task, taskErr := NewWaitForEventTask(raw, "http://localhost:8080", nil, nil)
	require.NoError(t, taskErr)
	api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString(), pluginState: string(notifyFailed)}
	task.Init(api)

	resp, err := task.GetRenderInfo(context.Background())
	require.NoError(t, err)

	content, ok := resp.Data.(GetRenderInfoResponse).Content.(map[string]any)
	require.True(t, ok)
	display, ok := content["display"].(*WaitForEventDisplay)
	require.True(t, ok)
	assert.Equal(t, "Sample Drop Off Confirmation", display.Title)
	assert.Equal(t, "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.", display.Description)
}

func TestWaitForEventTask_GetRenderInfo_DynamicDisplayMissingStateKey_FallsBackToEmptyDisplay(t *testing.T) {
	raw, err := json.Marshal(WaitForEventConfig{
		Submission: &SubmissionConfig{Url: "http://irrelevant"},
		Display: &WaitForEventDisplay{
			Title: map[string]any{
				"waiting": "Waiting on Sample Drop Off Confirmation",
				"failed":  "Sample Drop Off Confirmation Failed",
			},
			Description: map[string]any{
				"waiting": "Please drop off your sample at the designated location and confirm the drop off by clicking on this task",
				"failed":  "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.",
			},
		},
	})
	require.NoError(t, err)

	task, taskErr := NewWaitForEventTask(raw, "http://localhost:8080", nil, nil)
	require.NoError(t, taskErr)
	api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString(), pluginState: string(receivedCallback)}
	task.Init(api)

	resp, err := task.GetRenderInfo(context.Background())
	require.NoError(t, err)

	content, ok := resp.Data.(GetRenderInfoResponse).Content.(map[string]any)
	require.True(t, ok)
	display, ok := content["display"].(*WaitForEventDisplay)
	require.True(t, ok)
	assert.Nil(t, display.Title)
	assert.Nil(t, display.Description)
}

func TestWaitForEventTask_GetRenderInfo_UnknownPluginState_FallsBackToEmptyDisplay(t *testing.T) {
	raw, err := json.Marshal(WaitForEventConfig{
		Submission: &SubmissionConfig{Url: "http://irrelevant"},
		Display: &WaitForEventDisplay{
			Title: map[string]any{
				"waiting":   "Waiting on Sample Drop Off Confirmation",
				"failed":    "Sample Drop Off Confirmation Failed",
				"completed": "Sample Drop Off Confirmation Completed",
			},
			Description: map[string]any{
				"waiting":   "Please drop off your sample at the designated location and confirm the drop off by clicking on this task",
				"failed":    "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.",
				"completed": "We have received confirmation of your sample drop off. We will notify you once the test results are available.",
			},
		},
	})
	require.NoError(t, err)

	task, taskErr := NewWaitForEventTask(raw, "http://localhost:8080", nil, nil)
	require.NoError(t, taskErr)
	api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString(), pluginState: "UNKNOWN_STATE"}
	task.Init(api)

	resp, err := task.GetRenderInfo(context.Background())
	require.NoError(t, err)

	content, ok := resp.Data.(GetRenderInfoResponse).Content.(map[string]any)
	require.True(t, ok)
	display, ok := content["display"].(*WaitForEventDisplay)
	require.True(t, ok)
	assert.Nil(t, display.Title)
	assert.Nil(t, display.Description)
}

// ── Start edge cases ──────────────────────────────────────────────────────────

func TestWaitForEventTask_Start_EmptyURL(t *testing.T) {
	raw, _ := json.Marshal(WaitForEventConfig{
		Submission: &SubmissionConfig{
			Url: "",
		},
	})
	task, taskErr := NewWaitForEventTask(raw, "http://localhost:8080", nil, nil)
	require.NoError(t, taskErr)
	api := &wfeAPI{taskID: uuid.NewString(), workflowID: uuid.NewString()}
	task.Init(api)

	resp, err := task.Start(context.Background())

	require.ErrorContains(t, err, "submission url not configured")
	assert.Nil(t, resp)
	assert.Empty(t, api.transitionCalls, "no transition should occur when URL is missing")
}

func TestWaitForEventTask_Start_TransitionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(action string) bool { return action == FSMActionStart }
	api.transitionErr = errors.New("db unavailable")

	_, err := task.Start(context.Background())

	require.ErrorContains(t, err, "db unavailable")
}

// ── Execute error paths ───────────────────────────────────────────────────────

func TestWaitForEventTask_Execute_Complete_TransitionError(t *testing.T) {
	task, api := newWFETask(t, "http://irrelevant")
	api.transitionErr = errors.New("db unavailable")

	_, err := task.Execute(context.Background(), &ExecutionRequest{Action: waitForEventFSMComplete})

	require.ErrorContains(t, err, "db unavailable")
}

func TestWaitForEventTask_Execute_Retry_TransitionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.transitionErr = errors.New("db unavailable")

	_, err := task.Execute(context.Background(), &ExecutionRequest{Action: waitForEventFSMRetry})

	require.ErrorContains(t, err, "db unavailable")
	assert.True(t, api.calledWith(waitForEventFSMRetry), "transition should have been attempted")
}

// ── notifyExternalService branches ───────────────────────────────────────────

func TestWaitForEventTask_Notify_ContextCancelledBeforeSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(_ string) bool { return true }

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	resp, err := task.Start(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Failed to notify external service", resp.Message)
	assert.True(t, api.calledWith(waitForEventFSMStartFailed), "should still transition to NOTIFY_FAILED")
}

func TestWaitForEventTask_Notify_NonRetryable4xx(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest) // 400 — non-retryable
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(_ string) bool { return true }

	resp, err := task.Start(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "non-retryable 4xx should not be retried")
	assert.Equal(t, "Failed to notify external service", resp.Message)
	assert.True(t, api.calledWith(waitForEventFSMStartFailed))
}

// TestWaitForEventTask_Start_TransitionToNotifyFailedError covers the slog.ErrorContext branch
// inside Start where Transition(START_FAILED) itself returns an error after a failed notification.
func TestWaitForEventTask_Start_TransitionToNotifyFailedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // non-retryable so no retry delay
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(_ string) bool { return true }
	api.transitionErr = errors.New("db unavailable") // START_FAILED transition also fails

	resp, err := task.Start(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db unavailable")
	assert.Nil(t, resp)
	assert.True(t, api.calledWith(waitForEventFSMStartFailed), "START_FAILED transition should be attempted")
}

func TestWaitForEventTask_Notify_ContextCancelledDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // triggers retry backoff
		cancel()                                      // cancel immediately so ctx.Done() wins the backoff select
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(_ string) bool { return true }

	resp, err := task.Start(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Failed to notify external service", resp.Message)
	assert.True(t, api.calledWith(waitForEventFSMStartFailed))
}

func TestWaitForEventTask_Notify_TooManyRequests_IsRetried(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests) // 429 — retryable
	}))
	defer srv.Close()

	task, api := newWFETask(t, srv.URL)
	api.canTransition = func(_ string) bool { return true }

	resp, err := task.Start(context.Background())

	require.NoError(t, err)
	assert.Greater(t, callCount, 1, "429 should be retried")
	assert.Equal(t, "Failed to notify external service", resp.Message)
	assert.True(t, api.calledWith(waitForEventFSMStartFailed))
}
