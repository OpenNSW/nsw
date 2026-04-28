package persistence

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/OpenNSW/nsw/internal/task/plugin"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) Create(task *TaskWorkflowTask) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *mockStore) GetByTaskID(taskID string) (*TaskWorkflowTask, error) {
	args := m.Called(taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TaskWorkflowTask), args.Error(1)
}

func (m *mockStore) GetStateByTaskID(taskID string) (plugin.State, error) {
	args := m.Called(taskID)
	return args.Get(0).(plugin.State), args.Error(1)
}

func (m *mockStore) GetDataByTaskID(taskID string) (json.RawMessage, error) {
	args := m.Called(taskID)
	return args.Get(0).(json.RawMessage), args.Error(1)
}

func (m *mockStore) GetByMacroWorkflowID(macroWorkflowID string) ([]TaskWorkflowTask, error) {
	args := m.Called(macroWorkflowID)
	return args.Get(0).([]TaskWorkflowTask), args.Error(1)
}

func (m *mockStore) Update(task *TaskWorkflowTask) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *mockStore) UpdateState(taskID string, state plugin.State) error {
	args := m.Called(taskID, state)
	return args.Error(0)
}

func (m *mockStore) UpdateData(taskID string, data json.RawMessage) error {
	args := m.Called(taskID, data)
	return args.Error(0)
}

func (m *mockStore) Delete(taskID string) error {
	args := m.Called(taskID)
	return args.Error(0)
}

func TestNewTaskScopedStore(t *testing.T) {
	t.Run("rejects nil store", func(t *testing.T) {
		scoped, err := NewTaskScopedStore(nil, "task-1")

		assert.Nil(t, scoped)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "store cannot be nil")
	})

	t.Run("rejects empty taskID", func(t *testing.T) {
		scoped, err := NewTaskScopedStore(new(mockStore), "")

		assert.Nil(t, scoped)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "taskID cannot be empty")
	})

	t.Run("returns restricted interface", func(t *testing.T) {
		scoped, err := NewTaskScopedStore(new(mockStore), "task-1")

		assert.NoError(t, err)
		assert.NotNil(t, scoped)
	})
}

func TestTaskScopedStoreGet(t *testing.T) {
	store := new(mockStore)
	data := json.RawMessage(`{"form":"render"}`)
	store.On("GetByTaskID", "task-1").Return(&TaskWorkflowTask{
		TaskID:          "task-1",
		MacroWorkflowID: "macro-1",
		TaskTemplateID:  "template-1",
		State:           plugin.InProgress,
		Data:            data,
	}, nil).Once()

	scoped, err := NewTaskScopedStore(store, "task-1")
	assert.NoError(t, err)

	record, err := scoped.Get()

	assert.NoError(t, err)
	assert.Equal(t, &TaskScopedRecord{
		TaskID: "task-1",
		State:  plugin.InProgress,
		Data:   data,
	}, record)
	store.AssertExpectations(t)
}

func TestTaskScopedStoreGetState(t *testing.T) {
	store := new(mockStore)
	store.On("GetStateByTaskID", "task-1").Return(plugin.Completed, nil).Once()

	scoped, err := NewTaskScopedStore(store, "task-1")
	assert.NoError(t, err)

	state, err := scoped.GetState()

	assert.NoError(t, err)
	assert.Equal(t, plugin.Completed, state)
	store.AssertExpectations(t)
}

func TestTaskScopedStoreSetState(t *testing.T) {
	store := new(mockStore)
	store.On("UpdateState", "task-1", plugin.Failed).Return(nil).Once()

	scoped, err := NewTaskScopedStore(store, "task-1")
	assert.NoError(t, err)

	err = scoped.SetState(plugin.Failed)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestTaskScopedStoreGetData(t *testing.T) {
	store := new(mockStore)
	data := json.RawMessage(`{"field":"value"}`)
	store.On("GetDataByTaskID", "task-1").Return(data, nil).Once()

	scoped, err := NewTaskScopedStore(store, "task-1")
	assert.NoError(t, err)

	got, err := scoped.GetData()

	assert.NoError(t, err)
	assert.Equal(t, data, got)
	store.AssertExpectations(t)
}

func TestTaskScopedStoreSetData(t *testing.T) {
	store := new(mockStore)
	data := json.RawMessage(`{"field":"updated"}`)
	store.On("UpdateData", "task-1", data).Return(nil).Once()

	scoped, err := NewTaskScopedStore(store, "task-1")
	assert.NoError(t, err)

	err = scoped.SetData(data)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestTaskScopedStorePassesThroughErrors(t *testing.T) {
	store := new(mockStore)
	wantErr := errors.New("store failed")
	store.On("GetByTaskID", "task-1").Return(nil, wantErr).Once()

	scoped, err := NewTaskScopedStore(store, "task-1")
	assert.NoError(t, err)

	record, err := scoped.Get()

	assert.Nil(t, record)
	assert.ErrorIs(t, err, wantErr)
	store.AssertExpectations(t)
}
