package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPI is a mock implementation of the API interface for testing plugins
type MockAPI struct {
	mock.Mock
}

func (m *MockAPI) GetTaskID() uuid.UUID {
	args := m.Called()
	return args.Get(0).(uuid.UUID)
}

func (m *MockAPI) GetWorkflowID() uuid.UUID {
	args := m.Called()
	return args.Get(0).(uuid.UUID)
}

func (m *MockAPI) GetTaskState() State {
	args := m.Called()
	return args.Get(0).(State)
}

func (m *MockAPI) ReadFromGlobalStore(key string) (any, bool) {
	args := m.Called(key)
	return args.Get(0), args.Bool(1)
}

func (m *MockAPI) WriteToLocalStore(key string, value any) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockAPI) ReadFromLocalStore(key string) (any, error) {
	args := m.Called(key)
	return args.Get(0), args.Error(1)
}

func (m *MockAPI) GetPluginState() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAPI) CanTransition(action string) bool {
	args := m.Called(action)
	return args.Bool(0)
}

func (m *MockAPI) Transition(action string) error {
	args := m.Called(action)
	return args.Error(0)
}

func TestSimpleForm_Execute_SaveAsDraft(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockAPI := new(MockAPI)

		// Create SimpleForm with empty config for testing
		sf, err := NewSimpleForm(json.RawMessage(`{}`), nil, nil)
		assert.NoError(t, err)

		sf.Init(mockAPI)

		data := map[string]interface{}{"field1": "value1"}
		req := &ExecutionRequest{
			Action:  SimpleFormActionDraft,
			Content: data,
		}

		mockAPI.On("CanTransition", SimpleFormActionDraft).Return(true).Once()
		mockAPI.On("WriteToLocalStore", "trader:form", data).Return(nil).Once()
		mockAPI.On("Transition", SimpleFormActionDraft).Return(nil).Once()

		resp, err := sf.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.ApiResponse)
		assert.True(t, resp.ApiResponse.Success)

		mockAPI.AssertExpectations(t)
	})

	t.Run("WriteToLocalStore Failure", func(t *testing.T) {
		mockAPI := new(MockAPI)

		sf, err := NewSimpleForm(json.RawMessage(`{}`), nil, nil)
		assert.NoError(t, err)

		sf.Init(mockAPI)

		data := map[string]interface{}{"field1": "value1"}
		req := &ExecutionRequest{
			Action:  SimpleFormActionDraft,
			Content: data,
		}

		mockLocalStoreErr := errors.New("local store error")

		mockAPI.On("CanTransition", SimpleFormActionDraft).Return(true).Once()
		mockAPI.On("WriteToLocalStore", "trader:form", data).Return(mockLocalStoreErr).Once()
		// Transition shouldn't be called if WriteToLocalStore fails

		resp, err := sf.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Equal(t, mockLocalStoreErr, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.ApiResponse)
		assert.False(t, resp.ApiResponse.Success)
		assert.Equal(t, "SAVE_DRAFT_FAILED", resp.ApiResponse.Error.Code)

		mockAPI.AssertExpectations(t)
	})

	t.Run("Invalid Transition", func(t *testing.T) {
		mockAPI := new(MockAPI)

		sf, err := NewSimpleForm(json.RawMessage(`{}`), nil, nil)
		assert.NoError(t, err)

		sf.Init(mockAPI)

		data := map[string]interface{}{"field1": "value1"}
		req := &ExecutionRequest{
			Action:  SimpleFormActionDraft,
			Content: data,
		}

		mockAPI.On("CanTransition", SimpleFormActionDraft).Return(false).Once()
		mockAPI.On("GetPluginState").Return(string(TraderSubmitted)).Once()

		resp, err := sf.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted in state")
		assert.Nil(t, resp)

		mockAPI.AssertExpectations(t)
	})
}
