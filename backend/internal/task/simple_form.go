package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/google/uuid"
)

// SimpleFormAction represents the action to perform on the trader form
type SimpleFormAction string

const (
	SimpleFormActionFetch     SimpleFormAction = "FETCH_FORM"
	SimpleFormActionSubmit    SimpleFormAction = "SUBMIT_FORM"
	SimpleFormActionReject    SimpleFormAction = "REJECT_FORM"
	SimpleFormActionOgaVerify SimpleFormAction = "OGA_VERIFICATION"
)

// SimpleFormCommandSet contains the JSON Form configuration for the trader form
type SimpleFormCommandSet struct {
	FormID                  string          `json:"formId"`                            // Unique identifier for the form
	Title                   string          `json:"title"`                             // Display title of the form
	Schema                  json.RawMessage `json:"schema"`                            // JSON Schema defining the form structure and validation
	UISchema                json.RawMessage `json:"uiSchema,omitempty"`                // UI Schema for rendering hints (optional)
	FormData                json.RawMessage `json:"formData,omitempty"`                // Default/pre-filled form data (optional)
	SubmissionURL           string          `json:"submissionUrl,omitempty"`           // URL to submit form data to (optional)
	RequiresOgaVerification bool            `json:"requiresOgaVerification,omitempty"` // If true, waits for OGA_VERIFICATION action; if false, completes after submission response
}

// SimpleFormDefinition holds the complete form definition for a specific form
type SimpleFormDefinition struct {
	Title    string          `json:"title"`
	Schema   json.RawMessage `json:"schema"`
	UISchema json.RawMessage `json:"uiSchema,omitempty"`
	FormData json.RawMessage `json:"formData,omitempty"`
}

// SimpleFormResult extends ExecutionResult with form-specific response data
type SimpleFormResult struct {
	FormID   string          `json:"formId,omitempty"`
	Title    string          `json:"title,omitempty"`
	Schema   json.RawMessage `json:"schema,omitempty"`
	UISchema json.RawMessage `json:"uiSchema,omitempty"`
	FormData json.RawMessage `json:"formData,omitempty"`
}

type SimpleFormTask struct {
	config      *config.Config
	formService form.FormService
}

// NewSimpleFormTask creates a new SimpleFormTask
func NewSimpleFormTask(cfg *config.Config, formService form.FormService) (*SimpleFormTask, error) {
	return &SimpleFormTask{config: cfg, formService: formService}, nil
}

// Start initializes the task, fetches form definition, and prepopulates data
func (t *SimpleFormTask) Start(ctx context.Context, config json.RawMessage, is StateManager, gs StateManager) (*TaskPluginReturnValue, error) {
	commandSet, err := parseSimpleFormCommandSet(ctx, config, t.formService)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command set: %w", err)
	}

	is.Set("form", commandSet)

	// Return schema to frontend
	return &TaskPluginReturnValue{
		Status:                 TaskStatusSuspended,
		StatusHumanReadableStr: "Awaiting form submission",
		Data: map[string]any{
			"formId":   commandSet.FormID,
			"title":    commandSet.Title,
			"schema":   commandSet.Schema,
			"uiSchema": commandSet.UISchema,
			"formData": commandSet.FormData,
		},
	}, nil
}

func (t *SimpleFormTask) Resume(ctx context.Context, is StateManager, gs StateManager, data map[string]interface{}) (*TaskPluginReturnValue, error) {
	action, ok := data["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required in resume data")
	}

	// Re-construct commandSet from InternalState
	var commandSet SimpleFormCommandSet
	formState, ok := is.Get("form")
	if ok && formState != nil {
		// StateManager stores interface{}, so we need to re-marshal/unmarshal to get the struct
		bytes, err := json.Marshal(formState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal form state: %w", err)
		}
		if err := json.Unmarshal(bytes, &commandSet); err != nil {
			return nil, fmt.Errorf("failed to unmarshal form state: %w", err)
		}
	} else {
		return nil, fmt.Errorf("form definition missing from internal state")
	}

	// The user code separates actions.
	
	switch action {
	case string(SimpleFormActionSubmit):
		formData, ok := data["formData"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("formData is required for submission")
		}
		return t.handleSubmitForm(ctx, &commandSet, formData, is, gs)

	case string(SimpleFormActionOgaVerify):
		return t.handleOgaVerification(data, is)

	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// handleSubmitForm handles the SUBMIT_FORM action
func (t *SimpleFormTask) handleSubmitForm(ctx context.Context, commandSet *SimpleFormCommandSet, formData map[string]interface{}, is StateManager, gs StateManager) (*TaskPluginReturnValue, error) {
	// Here you would typically send the formData to the commandSet.SubmissionURL
	// For now, we'll just log it and proceed.
	slog.InfoContext(ctx, "Form submitted", "formId", commandSet.FormID, "formData", formData)

	if commandSet.RequiresOgaVerification {
		is.Set("awaiting_oga", true)
		return &TaskPluginReturnValue{
			Status:                 TaskStatusSuspended,
			StatusHumanReadableStr: "Awaiting OGA verification",
			Data:                   formData, // Optionally return submitted data
		}, nil
	}

	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: "Form submitted successfully",
		Data:                   formData,
	}, nil
}

// handleOgaVerification handles the OGA_VERIFICATION action
func (t *SimpleFormTask) handleOgaVerification(verificationData map[string]interface{}, is StateManager) (*TaskPluginReturnValue, error) {
	is.Set("awaiting_oga", false)
	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: "Verified by OGA",
		Data:                   verificationData,
	}, nil
}

// parseSimpleFormCommandSet parses the command set into SimpleFormCommandSet.
// If only formId is provided, it looks up the form definition from the registry.
func parseSimpleFormCommandSet(ctx context.Context, commandSet interface{}, formService form.FormService) (*SimpleFormCommandSet, error) {
	if commandSet == nil {
		return nil, fmt.Errorf("command set is nil")
	}

	var parsed SimpleFormCommandSet

	switch cs := commandSet.(type) {
	case *SimpleFormCommandSet:
		parsed = *cs
	case SimpleFormCommandSet:
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
	if parsed.FormID != "" && parsed.Schema == nil {
		if err := populateFromRegistry(ctx, &parsed, formService); err != nil {
			// Don't return error if registry fetch fails, maybe it's purely dynamic or intended to fail later
			slog.WarnContext(ctx, "failed to populate from registry", "formId", parsed.FormID, "error", err)
		}
	}

	return &parsed, nil
}

// populateFromRegistry fills in the form definition from the registry based on formId
func populateFromRegistry(ctx context.Context, cs *SimpleFormCommandSet, formService form.FormService) error {
	if formService == nil {
		return fmt.Errorf("form service is required to populate form definition")
	}

	// Parse form ID as UUID
	formUUID, err := uuid.Parse(cs.FormID)
	if err != nil {
		return fmt.Errorf("invalid form ID format (expected UUID): %w", err)
	}

	// Get form from service
	def, err := formService.GetFormByID(ctx, formUUID)
	if err != nil {
		return fmt.Errorf("failed to get form definition for formId %s: %w", cs.FormID, err)
	}

	cs.Title = def.Name
	cs.Schema = def.Schema
	cs.UISchema = def.UISchema
	// FormData is not stored in the form definition in the DB model currently,
	// so we leave it as is (might be populated from commandSet or pre-population logic)

	return nil
}
