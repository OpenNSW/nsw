package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
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

type SimpleFormTask struct {
	config *config.Config
	api    PluginAPI
}

// NewSimpleFormTask creates a new SimpleFormTask
func NewSimpleFormTask(cfg *config.Config, api PluginAPI) *SimpleFormTask {
	return &SimpleFormTask{config: cfg, api: api}
}

// Start initializes the task, fetches form definition, and prepopulates data
func (t *SimpleFormTask) Start(ctx context.Context, config map[string]any) (*TaskPluginReturnValue, error) {
	// Use API to parse Command Set (which involves form fetching)
	commandSet, err := parseSimpleFormCommandSet(ctx, config, t.api)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command set: %w", err)
	}

	// Populate form data from global context if applicable
	if err := populateFormDataFromGlobalContext(ctx, t.api, commandSet); err != nil {
		slog.WarnContext(ctx, "failed to populate form data from global context", "error", err)
	}

	// Store form definition in state via API
	if err := t.api.WriteToLocalStore("form", commandSet); err != nil {
		return nil, fmt.Errorf("failed to store form state: %w", err)
	}

	// Return schema to frontend
	data := map[string]any{
		"formId":   commandSet.FormID,
		"title":    commandSet.Title,
		"schema":   commandSet.Schema,
		"uiSchema": commandSet.UISchema,
		"formData": commandSet.FormData,
	}
	
	return &TaskPluginReturnValue{
		Status:                 TaskStatusAwaitingInput,
		StatusHumanReadableStr: string(TaskStatusAwaitingInput),
		Data:                   data,
	}, nil
}

func (t *SimpleFormTask) Resume(ctx context.Context, data map[string]any) (*TaskPluginReturnValue, error) {
	action, ok := data["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required in resume data")
	}

	// Re-construct commandSet from State via API
	var commandSet SimpleFormCommandSet
	formState, ok := t.api.ReadFromLocalStore("form")
	if ok && formState != nil {
		bytes, err := json.Marshal(formState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal form state: %w", err)
		}
		if err := json.Unmarshal(bytes, &commandSet); err != nil {
			return nil, fmt.Errorf("failed to unmarshal form state: %w", err)
		}
	} else {
		return nil, fmt.Errorf("form definition missing from state")
	}

	switch action {
	case string(SimpleFormActionSubmit):
		formData, ok := data["formData"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("formData is required for submission")
		}
		return t.handleSubmitForm(ctx, &commandSet, formData)

	case string(SimpleFormActionOgaVerify):
		return t.handleOgaVerification(data)

	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// handleSubmitForm handles the SUBMIT_FORM action
func (t *SimpleFormTask) handleSubmitForm(ctx context.Context, commandSet *SimpleFormCommandSet, formData map[string]interface{}) (*TaskPluginReturnValue, error) {
	// Here you would typically send the formData to the commandSet.SubmissionURL
	// For now, we'll just log it and proceed.
	slog.InfoContext(ctx, "Form submitted", "formId", commandSet.FormID, "formData", formData)

	if commandSet.RequiresOgaVerification {
		t.api.WriteToLocalStore("awaiting_oga", true)
		return &TaskPluginReturnValue{
			Status:                 TaskStatusAwaitingInput,
			StatusHumanReadableStr: string(TaskStatusAwaitingInput),
			Data:                   formData,
		}, nil
	}

	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: string(TaskStatusCompleted),
		Data:                   formData,
	}, nil
}

// handleOgaVerification handles the OGA_VERIFICATION action
func (t *SimpleFormTask) handleOgaVerification(verificationData map[string]interface{}) (*TaskPluginReturnValue, error) {
	t.api.WriteToLocalStore("awaiting_oga", false)
	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: string(TaskStatusCompleted),
		Data:                   verificationData,
	}, nil
}

// parseSimpleFormCommandSet parses the command set into SimpleFormCommandSet.
// If only formId is provided, it looks up the form definition from the registry.
func parseSimpleFormCommandSet(ctx context.Context, commandSet interface{}, api PluginAPI) (*SimpleFormCommandSet, error) {
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
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: "json",
			Result:  &parsed,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create decoder: %w", err)
		}
		if err := decoder.Decode(cs); err != nil {
			return nil, fmt.Errorf("failed to decode command set: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(cs, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command set from bytes: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported command set type: %T", commandSet)
	}

	// If only formId is provided, populate from registry
	if parsed.FormID != "" && parsed.Schema == nil {
		if err := populateFromRegistry(ctx, &parsed, api); err != nil {
			// Don't return error if registry fetch fails, maybe it's purely dynamic or intended to fail later
			slog.WarnContext(ctx, "failed to populate from registry", "formId", parsed.FormID, "error", err)
		}
	}

	return &parsed, nil
}

// populateFromRegistry fills in the form definition from the registry based on formId
func populateFromRegistry(ctx context.Context, cs *SimpleFormCommandSet, api PluginAPI) error {
	if api == nil {
		return fmt.Errorf("PluginAPI is required to populate form definition")
	}

	// Parse form ID as UUID
	formUUID, err := uuid.Parse(cs.FormID)
	if err != nil {
		return fmt.Errorf("invalid form ID format (expected UUID): %w", err)
	}

	// Get form from service via API
	def, err := api.GetFormById(ctx, formUUID)
	if err != nil {
		return fmt.Errorf("failed to get form definition for formId %s: %w", cs.FormID, err)
	}

	cs.Title = def.Name
	cs.Schema = def.Schema
	cs.UISchema = def.UISchema

	return nil
}

// populateFormDataFromGlobalContext traverses the schema and populates formData with values from the global store
// based on 'x-globalContext' annotations.
func populateFormDataFromGlobalContext(ctx context.Context, api PluginAPI, commandSet *SimpleFormCommandSet) error {
	if commandSet.Schema == nil {
		return nil
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(commandSet.Schema, &schemaMap); err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	var formDataMap map[string]interface{}
	// If FormData is present, unmarshal it. Otherwise create a new map.
	if len(commandSet.FormData) > 0 {
		if err := json.Unmarshal(commandSet.FormData, &formDataMap); err != nil {
			return fmt.Errorf("failed to unmarshal existing form data: %w", err)
		}
	} else {
		formDataMap = make(map[string]interface{})
	}

	// Walk schema and populate form data
	if err := walkSchemaAndPopulate(api, schemaMap, formDataMap); err != nil {
		return err
	}

	// Update commandSet.FormData with the mapped data
	newFormData, err := json.Marshal(formDataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated form data: %w", err)
	}
	commandSet.FormData = newFormData
	return nil
}

func walkSchemaAndPopulate(api PluginAPI, schema map[string]interface{}, data map[string]interface{}) error {
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil
	}

	for key, propVal := range props {
		propMap, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}

		// Handle nested objects
		if typeVal, ok := propMap["type"].(string); ok && typeVal == "object" {
			nestedData, _ := data[key].(map[string]interface{})
			if nestedData == nil {
				nestedData = make(map[string]interface{})
			}
			if err := walkSchemaAndPopulate(api, propMap, nestedData); err != nil {
				return err
			}
			if len(nestedData) > 0 {
				data[key] = nestedData
			}
			continue
		}

		// Check for x-globalContext annotation
		if globalKey, ok := propMap["x-globalContext"].(string); ok {
			if val, found := api.ReadFromGlobalStore(globalKey); found {
				data[key] = val
			}
		}
	}
	return nil
}
