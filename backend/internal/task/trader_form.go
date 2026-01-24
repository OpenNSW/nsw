package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/mocks"
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

// TraderFormDefinition holds the complete form definition for a specific form
type TraderFormDefinition struct {
	Title      string          `json:"title"`
	JSONSchema json.RawMessage `json:"jsonSchema"`
	UISchema   json.RawMessage `json:"uiSchema,omitempty"`
	FormData   json.RawMessage `json:"formData,omitempty"`
}

// getFormDefinition retrieves the complete form definition for a given form ID
func getFormDefinition(formID string) (*TraderFormDefinition, error) {
	filePath := fmt.Sprintf("forms/%s.json", formID)
	data, err := mocks.FS.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("form definition not found for formId: %s", formID)
	}

	var def TraderFormDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse form JSON: %w", err)
	}

	return &def, nil
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

// parseTraderFormCommandSet parses the command set into TraderFormCommandSet.
// If only formId is provided, it looks up the form definition from the registry.
func parseTraderFormCommandSet(commandSet interface{}) (*TraderFormCommandSet, error) {
	if commandSet == nil {
		return nil, fmt.Errorf("command set is nil")
	}

	var parsed TraderFormCommandSet

	switch cs := commandSet.(type) {
	case *TraderFormCommandSet:
		parsed = *cs
	case TraderFormCommandSet:
		parsed = cs
	case json.RawMessage:
		if err := json.Unmarshal(cs, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command set: %w", err)
		}
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(cs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command set: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command set: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported command set type: %T", commandSet)
	}

	// If only formId is provided, populate from registry
	if parsed.FormID != "" && parsed.JSONSchema == nil {
		if err := populateFromRegistry(&parsed); err != nil {
			return nil, err
		}
	}

	return &parsed, nil
}

// populateFromRegistry fills in the form definition from the registry based on formId
func populateFromRegistry(cs *TraderFormCommandSet) error {
	def, err := getFormDefinition(cs.FormID)
	if err != nil {
		return err
	}

	cs.Title = def.Title
	cs.JSONSchema = def.JSONSchema
	cs.UISchema = def.UISchema
	if cs.FormData == nil {
		cs.FormData = def.FormData
	}
	return nil
}

func (t *TraderFormTask) Execute(_ context.Context, payload *ExecutionPayload) (*ExecutionResult, error) {
	// Parse the payload to determine the action
	formPayload, err := t.parsePayload(payload)
	if err != nil {
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: fmt.Sprintf("Invalid payload: %v", err),
			Data:    formPayload,
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

// parsePayload parses the incoming ExecutionPayload into TraderFormPayload
func (t *TraderFormTask) parsePayload(payload *ExecutionPayload) (*TraderFormPayload, error) {
	if payload == nil {
		// Default to FETCH_FORM if no payload provided
		return &TraderFormPayload{Action: TraderFormActionFetch}, nil
	}

	// Map the Action from ExecutionPayload to TraderFormAction
	action := TraderFormAction(payload.Action)
	if action == "" {
		action = TraderFormActionFetch
	}

	// Parse FormData from payload.Payload if action is SUBMIT_FORM
	var formData map[string]interface{}
	if action == TraderFormActionSubmit && payload.Payload != nil {
		switch p := payload.Payload.(type) {
		case map[string]interface{}:
			formData = p
		default:
			// Try to marshal and unmarshal to get map[string]interface{}
			jsonBytes, err := json.Marshal(payload.Payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
			if err := json.Unmarshal(jsonBytes, &formData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}
	}

	return &TraderFormPayload{
		Action:   action,
		FormData: formData,
	}, nil
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
