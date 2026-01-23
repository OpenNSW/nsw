package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// TraderFormAction represents the action to perform on the trader form
type TraderFormAction string

const (
	TraderFormActionFetch  TraderFormAction = "FETCH_FORM"
	TraderFormActionSubmit TraderFormAction = "SUBMIT_FORM"
)

// TraderFormCommandSet contains the JSON Form configuration for the trader form
type TraderFormCommandSet struct {
	FormID     string          `json:"formId"`             // Unique identifier for the form
	Title      string          `json:"title"`              // Display title of the form
	JSONSchema json.RawMessage `json:"jsonSchema"`         // JSON Schema defining the form structure and validation
	UISchema   json.RawMessage `json:"uiSchema,omitempty"` // UI Schema for rendering hints (optional)
	FormData   json.RawMessage `json:"formData,omitempty"` // Default/pre-filled form data (optional)
}

// TraderFormPayload represents the payload for trader form actions
type TraderFormPayload struct {
	Action   TraderFormAction       `json:"action"`             // Action to perform: FETCH_FORM or SUBMIT_FORM
	FormData map[string]interface{} `json:"formData,omitempty"` // Form data for SUBMIT_FORM action
}

// TraderFormResult extends ExecutionResult with form-specific response data
type TraderFormResult struct {
	*ExecutionResult
	FormID     string          `json:"formId,omitempty"`
	Title      string          `json:"title,omitempty"`
	JSONSchema json.RawMessage `json:"jsonSchema,omitempty"`
	UISchema   json.RawMessage `json:"uiSchema,omitempty"`
	FormData   json.RawMessage `json:"formData,omitempty"`
}

type TraderFormTask struct {
	commandSet *TraderFormCommandSet
}

// NewTraderFormTask creates a new TraderFormTask with the provided command set.
// The commandSet can be of type *TraderFormCommandSet, TraderFormCommandSet,
// json.RawMessage, or map[string]interface{}.
func NewTraderFormTask(commandSet interface{}) (*TraderFormTask, error) {
	parsed, err := parseTraderFormCommandSet(commandSet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command set: %w", err)
	}
	return &TraderFormTask{commandSet: parsed}, nil
}

// parseTraderFormCommandSet parses the command set into TraderFormCommandSet
func parseTraderFormCommandSet(commandSet interface{}) (*TraderFormCommandSet, error) {
	if commandSet == nil {
		return nil, fmt.Errorf("command set is nil")
	}

	switch cs := commandSet.(type) {
	case *TraderFormCommandSet:
		return cs, nil
	case TraderFormCommandSet:
		return &cs, nil
	case json.RawMessage:
		var parsed TraderFormCommandSet
		if err := json.Unmarshal(cs, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command set: %w", err)
		}
		return &parsed, nil
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(cs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command set: %w", err)
		}
		var parsed TraderFormCommandSet
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command set: %w", err)
		}
		return &parsed, nil
	default:
		return nil, fmt.Errorf("unsupported command set type: %T", commandSet)
	}
}

func (t *TraderFormTask) Execute(_ context.Context, payload interface{}) (*ExecutionResult, error) {
	// Parse the payload to determine the action
	formPayload, err := t.parsePayload(payload)
	if err != nil {
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: fmt.Sprintf("Invalid payload: %v", err),
		}, err
	}

	// Handle action
	switch formPayload.Action {
	case TraderFormActionFetch:
		return t.handleFetchForm(t.commandSet)
	case TraderFormActionSubmit:
		return t.handleSubmitForm(t.commandSet, formPayload.FormData)
	default:
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: fmt.Sprintf("Unknown action: %s", formPayload.Action),
		}, fmt.Errorf("unknown action: %s", formPayload.Action)
	}
}

// parsePayload parses the incoming payload into TraderFormPayload
func (t *TraderFormTask) parsePayload(payload interface{}) (*TraderFormPayload, error) {
	if payload == nil {
		// Default to FETCH_FORM if no payload provided
		return &TraderFormPayload{Action: TraderFormActionFetch}, nil
	}

	switch p := payload.(type) {
	case *TraderFormPayload:
		return p, nil
	case TraderFormPayload:
		return &p, nil
	case map[string]interface{}:
		// Convert map to TraderFormPayload
		jsonBytes, err := json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		var formPayload TraderFormPayload
		if err := json.Unmarshal(jsonBytes, &formPayload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		return &formPayload, nil
	default:
		return nil, fmt.Errorf("unsupported payload type: %T", payload)
	}
}

// handleFetchForm returns the form schema for rendering
func (t *TraderFormTask) handleFetchForm(commandSet *TraderFormCommandSet) (*ExecutionResult, error) {
	// Return the form schema with READY status (task stays ready until form is submitted)
	return &ExecutionResult{
		Status:  model.TaskStatusReady,
		Message: "Form schema retrieved successfully",
		Data: TraderFormResult{
			FormID:     commandSet.FormID,
			Title:      commandSet.Title,
			JSONSchema: commandSet.JSONSchema,
			UISchema:   commandSet.UISchema,
			FormData:   commandSet.FormData,
		},
	}, nil
}

// handleSubmitForm validates and processes the form submission
func (t *TraderFormTask) handleSubmitForm(commandSet *TraderFormCommandSet, formData map[string]interface{}) (*ExecutionResult, error) {
	if formData == nil {
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: "Form data is required for submission",
		}, fmt.Errorf("form data is required for submission")
	}

	// TODO: Validate formData against JSONSchema
	// For now, we accept any valid JSON data

	// Convert formData to JSON for storage
	formDataJSON, err := json.Marshal(formData)
	if err != nil {
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: fmt.Sprintf("Failed to process form data: %v", err),
		}, err
	}

	// Return success with IN_PROGRESS status
	// The workflow manager will handle task state transitions
	return &ExecutionResult{
		Status:  model.TaskStatusInProgress,
		Message: "Trader form submitted successfully",
		Data: TraderFormResult{
			FormID:   commandSet.FormID,
			FormData: formDataJSON,
		},
	}, nil
}
