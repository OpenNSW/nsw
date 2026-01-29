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

func TestSimpleFormTask_FetchForm(t *testing.T) {
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

	commandSet := SimpleFormCommandSet{
		FormID: formID.String(),
	}

	// execute NewSimpleFormTask which should trigger populateFromRegistry
	task, err := NewSimpleFormTask(context.Background(), commandSet, nil, &config.Config{}, mockService)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Execute FETCH_FORM
	payload := &ExecutionPayload{
		Action: "FETCH_FORM",
	}

	result, err := task.Execute(context.Background(), payload)
	if err != nil {
		t.Fatalf("task execution failed: %v", err)
	}

	// Verify
	data, ok := result.Data.(SimpleFormResult)
	if !ok {
		t.Fatalf("expected SimpleFormResult, got %T", result.Data)
	}

	if data.Title != expectedTitle {
		t.Errorf("expected title %s, got %s", expectedTitle, data.Title)
	}
}
