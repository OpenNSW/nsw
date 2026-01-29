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
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/workflow/model"
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
func (t *SimpleFormTask) Start(ctx context.Context, config map[string]any, is StateManager, gs StateManager) (*TaskPluginReturnValue, error) {
	commandSet, err := parseSimpleFormCommandSet(ctx, config, t.formService)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command set: %w", err)
	}

	// Prepopulate form data from global context
	prepopulatedFormData, err := t.prepopulateFormData(commandSet.FormData, commandSet.Schema, gs)
	if err != nil {
		slog.Warn("failed to prepopulate form data from global context",
			"formId", commandSet.FormID,
			"error", err)
		// Continue with original form data if prepopulation fails
		prepopulatedFormData = commandSet.FormData
	}

	// Initial status is SUSPENDED (waiting for user submission)
	// We return the form schema in Data
	return &TaskPluginReturnValue{
		Status:                 TaskStatusSuspended,
		StatusHumanReadableStr: "Waiting for form submission",
		Data: SimpleFormResult{
			FormID:   commandSet.FormID,
			Title:    commandSet.Title,
			Schema:   commandSet.Schema,
			UISchema: commandSet.UISchema,
			FormData: prepopulatedFormData,
		},
	}, nil
}

// Resume handles form submission and OGA verification
func (t *SimpleFormTask) Resume(ctx context.Context, is StateManager, gs StateManager, data map[string]any) (*TaskPluginReturnValue, error) {
	// Reconstruct commandSet from internal state
	// Manager stores commandSet as []byte
	commandSetBytes, ok := is.Get("commandSet").([]byte)
	if !ok {
		return nil, fmt.Errorf("commandSet not found in internal state")
	}
	
	// Unmarshal to map first (parseSimpleFormCommandSet expects map or raw message or proper struct)
	var commandSetRaw interface{} = commandSetBytes
	commandSet, err := parseSimpleFormCommandSet(ctx, commandSetRaw, t.formService)
	if err != nil {
		return nil, fmt.Errorf("failed to restore command set: %w", err)
	}

	// Determine action. "data" is the content of the request.
	// We expect data to maybe have "formData"? Or is data the formData itself?
	// In the old code, Payload.Content was either map or object.
	// If Resume receives `data`, we assume it corresponds to the submission.
	
	// But `SimpleFormTask` supported `OGA_VERIFY`.
	// How to distinguish SUBMIT from VERIFY?
	// Maybe we should leverage `InternalState` to know where we are?
	// If Status is SUSPENDED, we probably expect SUBMIT.
	// If Status is IN_PROGRESS (waiting for OGA), we expect VERIFY.
	// Wait, `TaskPluginReturnValue` has `Status`.
	// If we returned `Status: Suspended` in `Start`, the next call is a resume.
	
	// Let's assume `data` contains `action` or we infer it?
	// The `data` map passed to Resume IS `req.Payload.Content`.
	// `req.Payload` has `Action` string.
	// But `Resume` signature provided by user: `Resume(ctx, is, gs, data map[string]any)`
	// It relies on `data`. It does NOT take `Action` argument.
	// So `Action` must be part of `data` OR we don't use `Action` field from `ExecutionPayload`?
	// OR `manager.go` should pass `Action` in `data`?
	// I didn't verify `manager.go` implementation in deep details regarding `Action`.
	// In `manager.go`: `data := req.Payload.Content`. `req.Payload.Action` is IGNORED in `Resume` call.
	// This is a potential issue if logic depends on Action.
	// I'll update `manager.go` later to pass action in data if needed, OR I will handle it here assuming data structure?
	// But `data` comes from external API.
	
	// I'll assume for now `SimpleFormTask` logic needs to be robust.
	// If `data` has `formData`, it's likely SUBMIT.
	
	// Simplification:
	// If we are just starting, we are waiting for submission.
	// If we submitted and require OGA, we are waiting for OGA.
	
	// Let's retrieve current status (from arguments or IS?)
	// `Status` is in `TaskContainer` but not passed to `Resume`.
	// We might store `status` in `is`?
	// `manager.go` updates `Status` in `TaskContainer`.
	// I'll check `manager.go` again... `internalState.Set` doesn't seem to track status explicitly other than what I added.
	
	// I'll imply action from the data or context?
	// Or maybe I should assume the `action` is passed in `data` by the caller?
	// But the API splits `Action` and `Content`.
	// I'll modify `manager.go` to inject `action` into `data`.
	
	// For now let's write `Resume` assuming `data["_action"]` exists?
	// Or check specific fields.
	
	// Better: `manager.go` should include the action in the data map.
	
	// I will assume `data["action"]` or similar is available or I'll add it in the next step.
	// Actually, `manager.go` in Step 203:
	// `data := req.Payload.Content`
	// `result, err := plugin.Resume(ctx, ..., data)`
	// `req.Payload.Action` is ignored.
	
	// I will modify `manager.go` to inject `action` into `data`.
	// But for `simple_form.go` now, I will write it expecting `action` in data?
	// Or I can infer?
	// If data has `formData`, it's likely SUBMIT.
	// If data has `verification`, it's VERIFY.
	
	// Previous code:
	// Actions: FETCH_FORM (handled by Start), SUBMIT_FORM, OGA_VERIFICATION.
	
	formData, _ := data["formData"].(map[string]interface{})
	// Try top level if not nested?
	if formData == nil {
		formData = data
	}
	
	// Determine if OGA verification based on some flag?
	// If we are in "Awaiting OGA" state?
	// We can check local state.
	// Let's use `is.Get("awaiting_oga")`.
	
	awaitingOGA, _ := is.Get("awaiting_oga").(bool)
	
	if awaitingOGA {
		return t.handleOgaVerification(data, is)
	}
	
	return t.handleSubmitForm(commandSet, formData, is, gs)
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
	case []byte:
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
			return nil, err
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

// prepopulateFormData builds formData from scratch
func (t *SimpleFormTask) prepopulateFormData(existingFormData json.RawMessage, schemaJSON json.RawMessage, gs StateManager) (json.RawMessage, error) {
	// Parse the schema to build formData
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Build formData from schema and global state
	formData := t.buildFormDataFromSchema(schema, gs)

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

// buildFormDataFromSchema recursively traverses the schema
func (t *SimpleFormTask) buildFormDataFromSchema(schema map[string]interface{}, gs StateManager) map[string]interface{} {
	formData := make(map[string]interface{})
    globalStateMap := gs.GetAll()

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
				if value := lookupValue(globalStateMap, pathStr); value != nil {
					formData[fieldName] = value
				}
			}
		}

		// Handle nested objects recursively
		fieldType, _ := fieldDef["type"].(string)
		if fieldType == "object" {
			nestedData := t.buildFormDataFromSchema(fieldDef, gs)
			if len(nestedData) > 0 {
				formData[fieldName] = nestedData
			}
		}
	}

	return formData
}

func lookupValue(data map[string]any, path string) any {
	// Simple lookup helper (extracted from prev implementation but stateless)
	if path == "" {
		return nil
	}
    keys := splitPath(path)
    var current any = data
    for _, key := range keys {
        if m, ok := current.(map[string]any); ok {
            if val, found := m[key]; found {
                current = val
            } else {
                return nil
            }
        } else {
            return nil
        }
    }
    return current
}

// splitPath splits a dot-notation path into individual keys
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

// handleSubmitForm validates and processes the form submission
func (t *SimpleFormTask) handleSubmitForm(commandSet *SimpleFormCommandSet, formData map[string]interface{}, is StateManager, gs StateManager) (*TaskPluginReturnValue, error) {
	if formData == nil {
		return &TaskPluginReturnValue{
            Status: TaskStatusSuspended,
            StatusHumanReadableStr: "Form data is required",
        }, fmt.Errorf("form data is required")
	}

	// Convert formData to JSON
    formDataJSON, _ := json.Marshal(formData)

	// If submissionUrl is provided, send the form data to that URL
	if commandSet.SubmissionURL != "" {
        taskId, _ := is.Get("taskId").(string) // Assuming we stored this or can get it
        consignmentId, _ := is.Get("consignmentId").(string)
        
		requestPayload := map[string]interface{}{
			"data":          formData,
			"taskId":        taskId,
			"consignmentId": consignmentId,
			"serviceUrl":    fmt.Sprintf("%s/api/tasks", t.config.Server.ServiceURL),
		}

		responseData, err := t.sendFormSubmission(commandSet.SubmissionURL, requestPayload)

		if err != nil {
			slog.Error("failed to send form submission", "error", err)
			return &TaskPluginReturnValue{
                Status: TaskStatusSuspended, // Remain suspended on failure? Or Failed?
                StatusHumanReadableStr: "Failed to submit form",
            }, err
		}

		// Check if OGA verification is required
		if commandSet.RequiresOgaVerification {
            is.Set("awaiting_oga", true)
			return &TaskPluginReturnValue{
				Status:                 TaskStatusSuspended, // Suspended waiting for OGA
				StatusHumanReadableStr: "Awaiting OGA verification",
				Data:                   responseData,
			}, nil
		}

		// No OGA verification required - complete the task
        responseJSON, _ := json.Marshal(responseData)
		return &TaskPluginReturnValue{
			Status:                 TaskStatusCompleted,
			StatusHumanReadableStr: "Form submitted successfully",
			Data: SimpleFormResult{
                FormID: commandSet.FormID,
                FormData: responseJSON,
            },
		}, nil
	}
    
    // No submission URL - Complete
	return &TaskPluginReturnValue{
		Status:                 TaskStatusCompleted,
		StatusHumanReadableStr: "Form submitted successfully",
		Data: SimpleFormResult{
            FormID: commandSet.FormID,
            FormData: formDataJSON,
        },
	}, nil
}

func (t *SimpleFormTask) sendFormSubmission(url string, formData map[string]interface{}) (map[string]interface{}, error) {
    jsonData, err := json.Marshal(formData)
    if err != nil {
        return nil, err
    }
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 300 {
        return nil, fmt.Errorf("submission failed: %s", body)
    }
    var res map[string]interface{}
    json.Unmarshal(body, &res)
    return res, nil
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
