package task

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/OpenNSW/nsw/internal/config"
	formmodel "github.com/OpenNSW/nsw/internal/form/model"
	"github.com/google/uuid"
)

// MockFormService is a mock implementation of FormService
type MockFormService struct {
	GetFormByIDFunc func(ctx context.Context, formID uuid.UUID) (*formmodel.FormResponse, error)
}

func (m *MockFormService) GetFormByID(ctx context.Context, formID uuid.UUID) (*formmodel.FormResponse, error) {
	if m.GetFormByIDFunc != nil {
		return m.GetFormByIDFunc(ctx, formID)
	}
	return nil, nil
}

// MockStateManager
type MockStateManager struct {
	data map[string]interface{}
}

func NewMockStateManager() *MockStateManager {
	return &MockStateManager{data: make(map[string]interface{})}
}

func (m *MockStateManager) Get(key string) (interface{}, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *MockStateManager) Set(key string, value interface{}) error {
	m.data[key] = value
	return nil
}

func (m *MockStateManager) GetAll() map[string]interface{} {
	return m.data
}

func TestSimpleFormTask_Start_FetchForm(t *testing.T) {
	// Setup
	formID := uuid.New()
	expectedTitle := "Test Form"
	expectedSchema := json.RawMessage(`{"type": "object"}`)

	mockService := &MockFormService{
		GetFormByIDFunc: func(ctx context.Context, id uuid.UUID) (*formmodel.FormResponse, error) {
			if id != formID {
				t.Errorf("expected form ID %s, got %s", formID, id)
			}
			return &formmodel.FormResponse{
				ID:     formID,
				Name:   expectedTitle,
				Schema: expectedSchema,
			}, nil
		},
	}

	// Configuration with FormID
	configMap := map[string]any{
		"formId": formID.String(),
	}
	configBytes, _ := json.Marshal(configMap)

	task, err := NewSimpleFormTask(&config.Config{}, mockService)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	is := NewMockStateManager()
	gs := NewMockStateManager()

	// Execute Start
	result, err := task.Start(context.Background(), json.RawMessage(configBytes), is, gs)
	if err != nil {
		t.Fatalf("task start failed: %v", err)
	}

	// Verify
	dataMap, ok := result.Data.(map[string]any)
	if !ok {
		// It returns map[string]any now in simple_form.go
		t.Fatalf("expected map[string]any, got %T", result.Data)
	}

	if dataMap["title"] != expectedTitle {
		t.Errorf("expected title %s, got %s", expectedTitle, dataMap["title"])
	}
	
	// Verify internal state
	formState, ok := is.Get("form")
	if !ok {
		t.Error("expected form state to be set")
	}
	if formState == nil {
		t.Error("expected form state to be non-nil")
	}
}
