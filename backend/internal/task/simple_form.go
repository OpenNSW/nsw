package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/workflow/model"
	"github.com/OpenNSW/nsw/mocks"
)

// SimpleFormAction represents the action to perform on the trader form
type SimpleFormAction string

const (
	SimpleFormActionFetch     SimpleFormAction = "FETCH_FORM"
	SimpleFormActionSubmit    SimpleFormAction = "SUBMIT_FORM"
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

// getFormDefinition retrieves the complete form definition for a given form ID
func getFormDefinition(formID string) (*SimpleFormDefinition, error) {
	filePath := fmt.Sprintf("forms/%s.json", formID)
	data, err := mocks.FS.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("form definition not found for formId: %s %w", formID, err)
	}

	var def SimpleFormDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse form JSON: %w", err)
	}

	return &def, nil
}

// SimpleFormPayload represents the payload for trader form actions
type SimpleFormPayload struct {
	Action   SimpleFormAction       `json:"action"`             // Action to perform: FETCH_FORM, SUBMIT_FORM, or OGA_VERIFICATION
	FormData map[string]interface{} `json:"formData,omitempty"` // Form data for SUBMIT_FORM action or verification data for OGA_VERIFICATION
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
	commandSet *SimpleFormCommandSet
	globalCtx  map[string]interface{}
	config     config.Config
}

// NewSimpleFormTask creates a new SimpleFormTask with the provided command set.
// The commandSet can be of type *SimpleFormCommandSet, SimpleFormCommandSet,
// json.RawMessage, or map[string]interface{}.
func NewSimpleFormTask(commandSet interface{}, globalCtx map[string]interface{}, cfg *config.Config) (*SimpleFormTask, error) {
	parsed, err := parseSimpleFormCommandSet(commandSet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command set: %w", err)
	}
	var config_ config.Config
	if cfg != nil {
		config_ = *cfg
	}
	return &SimpleFormTask{commandSet: parsed, globalCtx: globalCtx, config: config_}, nil
}

// parseSimpleFormCommandSet parses the command set into SimpleFormCommandSet.
// If only formId is provided, it looks up the form definition from the registry.
func parseSimpleFormCommandSet(commandSet interface{}) (*SimpleFormCommandSet, error) {
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
		if err := populateFromRegistry(&parsed); err != nil {
			return nil, err
		}
	}

	return &parsed, nil
}

// populateFromRegistry fills in the form definition from the registry based on formId
func populateFromRegistry(cs *SimpleFormCommandSet) error {
	def, err := getFormDefinition(cs.FormID)
	if err != nil {
		return err
	}

	cs.Title = def.Title
	cs.Schema = def.Schema
	cs.UISchema = def.UISchema
	if cs.FormData == nil {
		cs.FormData = def.FormData
	}
	return nil
}

// prepopulateFormData builds formData from scratch by traversing the schema
// and looking up values from globalCtx where x-globalContext is specified
func (t *SimpleFormTask) prepopulateFormData(existingFormData json.RawMessage) (json.RawMessage, error) {
	// If no global context, return existing form data as-is
	if len(t.globalCtx) == 0 {
		return existingFormData, nil
	}

	// Parse the schema to build formData
	var schema map[string]interface{}
	if err := json.Unmarshal(t.commandSet.Schema, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Build formData from schema
	formData := t.buildFormDataFromSchema(schema)

	// If we have existing formData, merge it (existing formData takes priority)
	if len(existingFormData) > 0 {
		var existingData map[string]interface{}
		if err := json.Unmarshal(existingFormData, &existingData); err == nil {
			formData = t.mergeFormData(formData, existingData)
		}
	}

	// If no data was populated, return nil to avoid sending empty object
	if len(formData) == 0 {
		return existingFormData, nil
	}

	// Convert to JSON
	prepopulatedJSON, err := json.Marshal(formData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prepopulated form data: %w", err)
	}

	return prepopulatedJSON, nil
}

// buildFormDataFromSchema recursively traverses the schema and builds formData
// by looking up values from globalCtx where x-globalContext is specified
func (t *SimpleFormTask) buildFormDataFromSchema(schema map[string]interface{}) map[string]interface{} {
	formData := make(map[string]interface{})

	// Get properties from schema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return formData
	}

	// Iterate through each property
	for fieldName, fieldDefRaw := range properties {
		fieldDef, ok := fieldDefRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if x-globalContext is specified
		if globalContextPath, exists := fieldDef["x-globalContext"]; exists {
			// Lookup value from global context
			if pathStr, ok := globalContextPath.(string); ok {
				if value := t.lookupValueFromGlobalContext(pathStr); value != nil {
					formData[fieldName] = value
				}
			}
		}

		// Handle nested objects recursively
		fieldType, _ := fieldDef["type"].(string)
		if fieldType == "object" {
			nestedData := t.buildFormDataFromSchema(fieldDef)
			if len(nestedData) > 0 {
				formData[fieldName] = nestedData
			}
		}

		// Handle arrays
		if fieldType == "array" {
			if items, ok := fieldDef["items"].(map[string]interface{}); ok {
				itemType, _ := items["type"].(string)
				if itemType == "object" {
					// For array of objects, we'd need actual data to know how many items
					// For now, we skip arrays - they typically need user input
					continue
				}
			}
		}
	}

	return formData
}

// lookupValueFromGlobalContext retrieves a value from globalCtx using dot notation path
// Example: "exporter.name" -> globalCtx["exporter"]["name"]
func (t *SimpleFormTask) lookupValueFromGlobalContext(path string) interface{} {
	if path == "" {
		return nil
	}

	// Split path by dots
	keys := splitPath(path)

	// Traverse the global context
	var current interface{} = t.globalCtx
	for _, key := range keys {
		if currentMap, ok := current.(map[string]interface{}); ok {
			var found bool
			current, found = currentMap[key]
			if !found {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

// splitPath splits a dot-notation path into individual keys
// Example: "exporter.name" -> ["exporter", "name"]
func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var keys []string
	start := 0
	for i, r := range path {
		if r == '.' {
			if i > start {
				keys = append(keys, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		keys = append(keys, path[start:])
	}

	return keys
}

// mergeFormData merges existing formData with prepopulated data
// Priority: existing > prepopulated (don't override explicit values)
func (t *SimpleFormTask) mergeFormData(prepopulated, existing map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy prepopulated data first
	for k, v := range prepopulated {
		result[k] = v
	}

	// Override with existing data
	for k, v := range existing {
		// If both are maps, merge recursively
		if existingMap, ok := v.(map[string]interface{}); ok {
			if prepopMap, ok := result[k].(map[string]interface{}); ok {
				result[k] = t.mergeFormData(prepopMap, existingMap)
				continue
			}
		}
		// Otherwise, existing takes priority
		result[k] = v
	}

	return result
}

func (t *SimpleFormTask) Execute(_ context.Context, payload *ExecutionPayload) (*ExecutionResult, error) {
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
	case SimpleFormActionFetch:
		return t.handleFetchForm(t.commandSet)
	case SimpleFormActionSubmit:
		return t.handleSubmitForm(t.commandSet, formPayload.FormData)
	case SimpleFormActionOgaVerify:
		return t.handleOgaVerification(formPayload.FormData)
	default:
		return &ExecutionResult{
			Message: fmt.Sprintf("Unknown action: %s", formPayload.Action),
		}, fmt.Errorf("unknown action: %s", formPayload.Action)
	}
}

// parsePayload parses the incoming ExecutionPayload into SimpleFormPayload
func (t *SimpleFormTask) parsePayload(payload *ExecutionPayload) (*SimpleFormPayload, error) {
	if payload == nil {
		// Default to FETCH_FORM if no payload provided
		return &SimpleFormPayload{Action: SimpleFormActionFetch}, nil
	}

	// Map the Action from ExecutionPayload to SimpleFormAction
	action := SimpleFormAction(payload.Action)
	if action == "" {
		action = SimpleFormActionFetch
	}

	// Parse FormData from payload.Content if action is SUBMIT_FORM or OGA_VERIFICATION
	var formData map[string]interface{}
	if (action == SimpleFormActionSubmit || action == SimpleFormActionOgaVerify) && payload.Content != nil {
		switch p := payload.Content.(type) {
		case map[string]interface{}:
			formData = p
		default:
			// Try to marshal and unmarshal to get map[string]interface{}
			jsonBytes, err := json.Marshal(payload.Content)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
			if err := json.Unmarshal(jsonBytes, &formData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}
	}

	return &SimpleFormPayload{
		Action:   action,
		FormData: formData,
	}, nil
}

// handleFetchForm returns the form schema for rendering
func (t *SimpleFormTask) handleFetchForm(commandSet *SimpleFormCommandSet) (*ExecutionResult, error) {
	// Prepopulate form data from global context
	prepopulatedFormData, err := t.prepopulateFormData(commandSet.FormData)
	if err != nil {
		slog.Warn("failed to prepopulate form data from global context",
			"formId", commandSet.FormID,
			"error", err)
		// Continue with original form data if prepopulation fails
		prepopulatedFormData = commandSet.FormData
	}

	// Return the form schema with READY status (task stays ready until form is submitted)
	return &ExecutionResult{
		Message: "Form schema retrieved successfully",
		Data: SimpleFormResult{
			FormID:   commandSet.FormID,
			Title:    commandSet.Title,
			Schema:   commandSet.Schema,
			UISchema: commandSet.UISchema,
			FormData: prepopulatedFormData,
		},
	}, nil
}

// handleSubmitForm validates and processes the form submission
func (t *SimpleFormTask) handleSubmitForm(commandSet *SimpleFormCommandSet, formData map[string]interface{}) (*ExecutionResult, error) {
	if formData == nil {
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: "Form data is required for submission",
		}, fmt.Errorf("form data is required for submission")
	}

	// TODO: Validate formData against Schema
	// For now, we accept any valid JSON data

	// Convert formData to JSON for storage
	formDataJSON, err := json.Marshal(formData)
	if err != nil {
		return &ExecutionResult{
			Status:  model.TaskStatusReady,
			Message: fmt.Sprintf("Failed to process form data: %v", err),
		}, err
	}

	// If submissionUrl is provided, send the form data to that URL
	if commandSet.SubmissionURL != "" {
		requestPayload := map[string]interface{}{
			"data":          formData,
			"taskId":        t.globalCtx["taskId"].(string),
			"consignmentId": t.globalCtx["consignmentId"].(string),
			"serviceUrl":    fmt.Sprintf("%s/api/tasks", t.config.Server.ServiceURL), // Include service URL from config + "/api/tasks"
		}

		responseData, err := t.sendFormSubmission(commandSet.SubmissionURL, requestPayload)

		if err != nil {
			slog.Error("failed to send form submission",
				"formId", commandSet.FormID,
				"submissionUrl", commandSet.SubmissionURL,
				"error", err)
			return &ExecutionResult{
				Status:  model.TaskStatusReady,
				Message: fmt.Sprintf("Failed to submit form to external system: %v", err),
			}, err
		}

		// Check if OGA verification is required
		if commandSet.RequiresOgaVerification {
			// Wait for OGA_VERIFICATION action to complete the task
			slog.Info("form submitted, waiting for OGA verification",
				"formId", commandSet.FormID,
				"submissionUrl", commandSet.SubmissionURL)

			return &ExecutionResult{
				Status:  model.TaskStatusInProgress,
				Message: "Form submitted successfully, awaiting OGA verification",
				Data: SimpleFormResult{
					FormID:   commandSet.FormID,
					FormData: formDataJSON,
				},
				GlobalContextData: formData,
			}, nil
		}

		// No OGA verification required - complete the task with response data
		responseJSON, err := json.Marshal(responseData)
		if err != nil {
			slog.Error("failed to marshal response data from submission", "formId", commandSet.FormID, "submissionUrl", commandSet.SubmissionURL, "error", err)
			return &ExecutionResult{
				Status:  model.TaskStatusReady,
				Message: fmt.Sprintf("Failed to process submission response: %v", err),
			}, err
		}
		slog.Info("form submitted and completed",
			"formId", commandSet.FormID,
			"submissionUrl", commandSet.SubmissionURL,
			"response", responseData)

		return &ExecutionResult{
			Status:  model.TaskStatusCompleted,
			Message: "Form submitted and processed successfully",
			Data: SimpleFormResult{
				FormID:   commandSet.FormID,
				FormData: responseJSON, // Store the response from submission
			},
			GlobalContextData: formData,
		}, nil
	}

	// If no submissionUrl, task moves to IN_PROGRESS
	return &ExecutionResult{
		Status:  model.TaskStatusCompleted,
		Message: "Trader form submitted successfully",
		Data: SimpleFormResult{
			FormID:   commandSet.FormID,
			FormData: formDataJSON,
		},
		GlobalContextData: formData,
	}, nil
}

// sendFormSubmission sends the form data to the specified URL via HTTP POST and returns the response
func (t *SimpleFormTask) sendFormSubmission(url string, formData map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(formData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal form data: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send POST request
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response JSON
	var responseData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &responseData); err != nil {
			slog.Warn("failed to parse response as JSON, storing as raw string",
				"url", url,
				"error", err)
			responseData = map[string]interface{}{
				"raw_response": string(body),
			}
		}
	}

	slog.Info("form submitted successfully",
		"url", url,
		"status", resp.StatusCode,
		"response", responseData)

	return responseData, nil
}

// handleOgaVerification handles the OGA_VERIFICATION action and marks the task as completed
func (t *SimpleFormTask) handleOgaVerification(verificationData map[string]interface{}) (*ExecutionResult, error) {
	// Convert verification data to JSON if present
	var verificationJSON json.RawMessage
	if verificationData != nil {
		jsonData, err := json.Marshal(verificationData)
		if err != nil {
			return &ExecutionResult{
				Status:  model.TaskStatusInProgress,
				Message: fmt.Sprintf("Failed to process verification data: %v", err),
			}, err
		}
		verificationJSON = jsonData
	}

	// Mark task as COMPLETED
	return &ExecutionResult{
		Status:  model.TaskStatusCompleted,
		Message: "Form verified by OGA, task completed",
		Data: SimpleFormResult{
			FormID:   t.commandSet.FormID,
			FormData: verificationJSON,
		},
	}, nil
}
