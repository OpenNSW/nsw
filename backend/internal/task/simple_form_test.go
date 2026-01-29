package task

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/OpenNSW/nsw/internal/config"
	formmodel "github.com/OpenNSW/nsw/internal/form/model"
	"github.com/google/uuid"
)

// MockPluginAPI is a mock implementation of PluginAPI
type MockPluginAPI struct {
	GetFormByIdFunc       func(ctx context.Context, formId uuid.UUID) (*formmodel.FormResponse, error)
	WriteToLocalStoreFunc func(key string, value interface{}) error
	ReadFromLocalStoreFunc func(key string) (interface{}, bool)
	WriteToGlobalStoreFunc func(key string, value interface{}) error
	ReadFromGlobalStoreFunc func(key string) (interface{}, bool)
	
	// Internal storage for default implementation
	localStorage map[string]interface{}
}

func NewMockPluginAPI() *MockPluginAPI {
	return &MockPluginAPI{
		localStorage: make(map[string]interface{}),
	}
}

func (m *MockPluginAPI) GetFormById(ctx context.Context, formId uuid.UUID) (*formmodel.FormResponse, error) {
	if m.GetFormByIdFunc != nil {
		return m.GetFormByIdFunc(ctx, formId)
	}
	return nil, nil
}

func (m *MockPluginAPI) WriteToLocalStore(key string, value interface{}) error {
	if m.WriteToLocalStoreFunc != nil {
		return m.WriteToLocalStoreFunc(key, value)
	}
	m.localStorage[key] = value
	return nil
}

func (m *MockPluginAPI) ReadFromLocalStore(key string) (interface{}, bool) {
	if m.ReadFromLocalStoreFunc != nil {
		return m.ReadFromLocalStoreFunc(key)
	}
	val, ok := m.localStorage[key]
	return val, ok
}

func (m *MockPluginAPI) WriteToGlobalStore(key string, value interface{}) error {
	if m.WriteToGlobalStoreFunc != nil {
		return m.WriteToGlobalStoreFunc(key, value)
	}
	return nil
}

func (m *MockPluginAPI) ReadFromGlobalStore(key string) (interface{}, bool) {
	if m.ReadFromGlobalStoreFunc != nil {
		return m.ReadFromGlobalStoreFunc(key)
	}
	return nil, false
}

func TestSimpleFormTask_Start_FetchForm(t *testing.T) {
	// Setup
	formID := uuid.New()
	expectedTitle := "Test Form"
	expectedSchema := json.RawMessage(`{"type": "object"}`)

	mockAPI := NewMockPluginAPI()
	mockAPI.GetFormByIdFunc = func(ctx context.Context, id uuid.UUID) (*formmodel.FormResponse, error) {
		if id != formID {
			t.Errorf("expected form ID %s, got %s", formID, id)
		}
		return &formmodel.FormResponse{
			ID:     formID,
			Name:   expectedTitle,
			Schema: expectedSchema,
		}, nil
	}

	// Configuration with FormID
	configMap := map[string]any{
		"formId": formID.String(),
	}

	task := NewSimpleFormTask(&config.Config{}, mockAPI)

	// Execute Start
	result, err := task.Start(context.Background(), configMap)
	if err != nil {
		t.Fatalf("task start failed: %v", err)
	}

	if result.Status != TaskStatusAwaitingInput {
		t.Errorf("expected status %s, got %s", TaskStatusAwaitingInput, result.Status)
	}

	// Verify Data
	dataMap, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any data, got %T", result.Data)
	}

	if dataMap["title"] != expectedTitle {
		t.Errorf("expected title %s, got %s", expectedTitle, dataMap["title"])
	}
	
	// Verify internal state via MockAPI (which simulates LocalStore)
	formState, ok := mockAPI.ReadFromLocalStore("form")
	if !ok {
		t.Error("expected form state to be set in local store")
	}
	if formState == nil {
		t.Error("expected form state to be non-nil")
	}
}

func TestSimpleFormTask_Start_Prepopulate(t *testing.T) {
	// Setup
	formID := uuid.New()
	globalKey := "someGlobalValue"
	expectedValue := "populatedValue"
	
	// properties with x-globalContext
	schemaJSON := fmt.Sprintf(`{
		"type": "object",
		"properties": {
			"myField": {
				"type": "string",
				"x-globalContext": "%s"
			}
		}
	}`, globalKey)
	
	expectedSchema := json.RawMessage(schemaJSON)

	mockAPI := NewMockPluginAPI()
	mockAPI.GetFormByIdFunc = func(ctx context.Context, id uuid.UUID) (*formmodel.FormResponse, error) {
		return &formmodel.FormResponse{
			ID:     formID,
			Name:   "Test Form",
			Schema: expectedSchema,
		}, nil
	}
	
	// Set value in global store
	mockAPI.ReadFromGlobalStoreFunc = func(key string) (interface{}, bool) {
		if key == globalKey {
			return expectedValue, true
		}
		return nil, false
	}

	configMap := map[string]any{
		"formId": formID.String(),
	}

	task := NewSimpleFormTask(&config.Config{}, mockAPI)

	// Execute Start
	result, err := task.Start(context.Background(), configMap)
	if err != nil {
		t.Fatalf("task start failed: %v", err)
	}

	// Verify Data
	dataMap, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any data")
	}
	
	// commandSet.FormData is json.RawMessage ([]byte)
	rawFormData, ok := dataMap["formData"].(json.RawMessage)
	if !ok {
		// try []byte
		bytes, ok := dataMap["formData"].([]byte)
		if !ok {
			// If it's nil, that's a failure (we expect it populated)
			if dataMap["formData"] == nil {
				t.Fatal("expected formData to be populated, got nil")
			}
			t.Fatalf("expected []byte or json.RawMessage for formData, got %T", dataMap["formData"])
		}
		rawFormData = json.RawMessage(bytes)
	}

	var formDataMap map[string]interface{}
	if err := json.Unmarshal(rawFormData, &formDataMap); err != nil {
		t.Fatalf("failed to unmarshal formData: %v", err)
	}
	
	if val, ok := formDataMap["myField"]; !ok || val != expectedValue {
		t.Errorf("expected myField to be %s, got %v", expectedValue, val)
	}
}
