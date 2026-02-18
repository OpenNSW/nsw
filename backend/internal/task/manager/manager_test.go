package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockTaskFactory
type MockTaskFactory struct {
	mock.Mock
}

func (m *MockTaskFactory) BuildExecutor(ctx context.Context, taskType plugin.Type, config json.RawMessage) (plugin.Plugin, error) {
	args := m.Called(ctx, taskType, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(plugin.Plugin), args.Error(1)
}

// MockTaskStore
type MockTaskStore struct {
	mock.Mock
}

func (m *MockTaskStore) Create(taskInfo *persistence.TaskInfo) error {
	args := m.Called(taskInfo)
	return args.Error(0)
}

func (m *MockTaskStore) GetByID(id uuid.UUID) (*persistence.TaskInfo, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*persistence.TaskInfo), args.Error(1)
}

func (m *MockTaskStore) UpdateStatus(id uuid.UUID, status *plugin.State) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockTaskStore) GetByWorkflowID(workflowID uuid.UUID) ([]persistence.TaskInfo, error) {
	args := m.Called(workflowID)
	return args.Get(0).([]persistence.TaskInfo), args.Error(1)
}

func (m *MockTaskStore) Update(taskInfo *persistence.TaskInfo) error {
	args := m.Called(taskInfo)
	return args.Error(0)
}

func (m *MockTaskStore) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockTaskStore) GetAll() ([]persistence.TaskInfo, error) {
	args := m.Called()
	return args.Get(0).([]persistence.TaskInfo), args.Error(1)
}

func (m *MockTaskStore) GetByStatus(status plugin.State) ([]persistence.TaskInfo, error) {
	args := m.Called(status)
	return args.Get(0).([]persistence.TaskInfo), args.Error(1)
}

func (m *MockTaskStore) UpdateLocalState(id uuid.UUID, localState json.RawMessage) error {
	args := m.Called(id, localState)
	return args.Error(0)
}

func (m *MockTaskStore) GetLocalState(id uuid.UUID) (json.RawMessage, error) {
	args := m.Called(id)
	return args.Get(0).(json.RawMessage), args.Error(1)
}

func (m *MockTaskStore) UpdatePluginState(id uuid.UUID, pluginState string) error {
	args := m.Called(id, pluginState)
	return args.Error(0)
}

func (m *MockTaskStore) GetPluginState(id uuid.UUID) (string, error) {
	args := m.Called(id)
	return args.Get(0).(string), args.Error(1)
}

// MockPlugin
type MockPlugin struct {
	mock.Mock
}

func (m *MockPlugin) Init(api plugin.API) {
	m.Called(api)
}

func (m *MockPlugin) Start(ctx context.Context) (*plugin.ExecutionResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.ExecutionResponse), args.Error(1)
}

func (m *MockPlugin) Execute(ctx context.Context, request *plugin.ExecutionRequest) (*plugin.ExecutionResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.ExecutionResponse), args.Error(1)
}

func (m *MockPlugin) GetRenderInfo(ctx context.Context) (*plugin.ApiResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plugin.ApiResponse), args.Error(1)
}

func TestInitTask(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}
		ctx := context.Background()
		taskID := uuid.New()
		req := InitTaskRequest{
			TaskID:                 taskID,
			WorkflowID:             uuid.New(),
			WorkflowNodeTemplateID: uuid.New(),
			Type:                   plugin.TaskTypeSimpleForm,
			Config:                 json.RawMessage(`{}`),
			GlobalState:            map[string]any{},
		}

		mockFactory.On("BuildExecutor", ctx, req.Type, req.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", req.TaskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", req.TaskID).Return("", nil).Once()
		mockStore.On("Create", mock.AnythingOfType("*persistence.TaskInfo")).Return(nil).Once()

		mockPlugin.On("Init", mock.Anything).Return().Once()

		state := plugin.InProgress
		resp := &plugin.ExecutionResponse{
			NewState: &state,
		}
		mockPlugin.On("Start", ctx).Return(resp, nil).Once()

		result, err := tm.InitTask(ctx, req)
		assert.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("BuildExecutor Error", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}
		ctx := context.Background()
		req := InitTaskRequest{
			TaskID: uuid.New(),
			Type:   plugin.TaskTypeSimpleForm,
			Config: json.RawMessage(`{}`),
		}

		mockFactory.On("BuildExecutor", ctx, req.Type, req.Config).Return(nil, errors.New("build error")).Once()

		result, err := tm.InitTask(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "build error")
	})

	t.Run("Plugin Start Error", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}
		ctx := context.Background()
		req := InitTaskRequest{
			TaskID: uuid.New(),
			Type:   plugin.TaskTypeSimpleForm,
			Config: json.RawMessage(`{}`),
		}

		mockFactory.On("BuildExecutor", ctx, req.Type, req.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", req.TaskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", req.TaskID).Return("", nil).Once()
		mockPlugin.On("Init", mock.Anything).Return().Once()

		// Store.Create called before Start
		mockStore.On("Create", mock.AnythingOfType("*persistence.TaskInfo")).Return(nil).Once()

		mockPlugin.On("Start", ctx).Return(nil, errors.New("start error")).Once()

		result, err := tm.InitTask(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "start error")
	})

	t.Run("Store Create Error", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}
		ctx := context.Background()
		req := InitTaskRequest{
			TaskID: uuid.New(),
			Type:   plugin.TaskTypeSimpleForm,
			Config: json.RawMessage(`{}`),
		}

		mockFactory.On("BuildExecutor", ctx, req.Type, req.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", req.TaskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", req.TaskID).Return("", nil).Once()
		mockPlugin.On("Init", mock.Anything).Return().Once()

		// Start NOT called if Create fails
		mockStore.On("Create", mock.AnythingOfType("*persistence.TaskInfo")).Return(errors.New("db error")).Once()

		result, err := tm.InitTask(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestHandleExecuteTask(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}

		taskID := uuid.New()
		workflowID := uuid.New()

		reqBody := ExecuteTaskRequest{
			WorkflowID: workflowID,
			TaskID:     taskID,
			Payload:    &plugin.ExecutionRequest{Action: "submit"},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Mock GetTask
		taskInfo := &persistence.TaskInfo{
			ID:                     taskID,
			WorkflowID:             workflowID,
			WorkflowNodeTemplateID: uuid.New(),
			Type:                   plugin.TaskTypeSimpleForm,
			Config:                 json.RawMessage(`{}`),
			GlobalContext:          json.RawMessage(`{}`),
		}
		mockStore.On("GetByID", taskID).Return(taskInfo, nil).Once()
		mockFactory.On("BuildExecutor", mock.Anything, taskInfo.Type, taskInfo.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", taskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", taskID).Return("", nil).Once()
		mockPlugin.On("Init", mock.Anything).Return().Once()

		// Mock Execute
		newState := plugin.Completed
		execResp := &plugin.ExecutionResponse{
			NewState:    &newState,
			ApiResponse: &plugin.ApiResponse{Success: true},
		}
		mockPlugin.On("Execute", mock.Anything, reqBody.Payload).Return(execResp, nil).Once()
		mockStore.On("UpdateStatus", taskID, &newState).Return(nil).Once()

		tm.HandleExecuteTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Execute Error", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}

		taskID := uuid.New()
		workflowID := uuid.New()

		reqBody := ExecuteTaskRequest{
			WorkflowID: workflowID,
			TaskID:     taskID,
			Payload:    &plugin.ExecutionRequest{Action: "submit"},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Mock GetTask
		taskInfo := &persistence.TaskInfo{
			ID:     taskID,
			Type:   plugin.TaskTypeSimpleForm,
			Config: json.RawMessage(`{}`),
		}
		mockStore.On("GetByID", taskID).Return(taskInfo, nil).Once()
		mockFactory.On("BuildExecutor", mock.Anything, taskInfo.Type, taskInfo.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", taskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", taskID).Return("", nil).Once()
		mockPlugin.On("Init", mock.Anything).Return().Once()

		// Mock Execute Error
		mockPlugin.On("Execute", mock.Anything, reqBody.Payload).Return(nil, errors.New("exec error")).Once()

		tm.HandleExecuteTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Invalid Method", func(t *testing.T) {
		tm := &taskManager{}
		req := httptest.NewRequest(http.MethodGet, "/execute", nil)
		w := httptest.NewRecorder()

		tm.HandleExecuteTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid Body", func(t *testing.T) {
		tm := &taskManager{}
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		tm.HandleExecuteTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHandleGetTask(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}

		taskID := uuid.New()
		workflowID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/tasks/"+taskID.String(), nil)
		req.SetPathValue("id", taskID.String())
		w := httptest.NewRecorder()

		// Mock GetTask (cache miss -> rebuild)
		taskInfo := &persistence.TaskInfo{
			ID:                     taskID,
			WorkflowID:             workflowID,
			WorkflowNodeTemplateID: uuid.New(),
			Type:                   plugin.TaskTypeSimpleForm,
			Config:                 json.RawMessage(`{}`),
			GlobalContext:          json.RawMessage(`{}`),
		}
		mockStore.On("GetByID", taskID).Return(taskInfo, nil).Once()
		mockFactory.On("BuildExecutor", mock.Anything, taskInfo.Type, taskInfo.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", taskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", taskID).Return("", nil).Once()

		mockPlugin.On("Init", mock.Anything).Return().Once()

		renderInfo := &plugin.ApiResponse{
			Success: true,
			Data:    map[string]string{"foo": "bar"},
		}
		mockPlugin.On("GetRenderInfo", mock.Anything).Return(renderInfo, nil).Once()

		tm.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result plugin.ApiResponse
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result.Success)
	})

	t.Run("GetRenderInfo Error", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		mockPlugin := new(MockPlugin)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}

		taskID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+taskID.String(), nil)
		req.SetPathValue("id", taskID.String())
		w := httptest.NewRecorder()

		// Mock GetTask
		taskInfo := &persistence.TaskInfo{
			ID:     taskID,
			Type:   plugin.TaskTypeSimpleForm,
			Config: json.RawMessage(`{}`),
		}
		mockStore.On("GetByID", taskID).Return(taskInfo, nil).Once()
		mockFactory.On("BuildExecutor", mock.Anything, taskInfo.Type, taskInfo.Config).Return(mockPlugin, nil).Once()
		mockStore.On("GetLocalState", taskID).Return(json.RawMessage(`{}`), nil).Once()
		mockStore.On("GetPluginState", taskID).Return("", nil).Once()
		mockPlugin.On("Init", mock.Anything).Return().Once()

		mockPlugin.On("GetRenderInfo", mock.Anything).Return(nil, errors.New("render error")).Once()

		tm.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Task Not Found", func(t *testing.T) {
		mockFactory := new(MockTaskFactory)
		mockStore := new(MockTaskStore)
		cfg := &config.Config{}
		tm := &taskManager{
			factory:        mockFactory,
			store:          mockStore,
			config:         cfg,
			containerCache: newContainerCache(10),
		}
		taskID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/tasks/"+taskID.String(), nil)
		req.SetPathValue("id", taskID.String())
		w := httptest.NewRecorder()

		mockStore.On("GetByID", taskID).Return(nil, gorm.ErrRecordNotFound).Once()

		tm.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Invalid Task ID", func(t *testing.T) {
		tm := &taskManager{}
		req := httptest.NewRequest(http.MethodGet, "/tasks/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		tm.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
