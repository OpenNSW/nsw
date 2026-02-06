package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/form"
)

// SimpleFormAction represents the action to perform on the form
const (
	SimpleFormActionFetch     = "FETCH_FORM"
	SimpleFormActionSubmit    = "SUBMIT_FORM"
	SimpleFormActionOgaVerify = "OGA_VERIFICATION"
)

// SimpleFormState represents the current state the form is in
type SimpleFormState string

const (
	Initialized        SimpleFormState = "INITIALIZED"
	TraderSavedAsDraft SimpleFormState = "DRAFT"
	TraderSubmitted    SimpleFormState = "SUBMITTED"
	OGAAcknowledged    SimpleFormState = "OGA_ACKNOWLEDGED"
	OGAReviewed        SimpleFormState = "OGA_REVIEWED"
)

// Config contains the JSON Form configuration
type Config struct {
	FormID                  string          `json:"formId"`                            // Unique identifier for the form
	Title                   string          `json:"title"`                             // Display title of the form
	Schema                  json.RawMessage `json:"schema"`                            // JSON Schema defining the form structure and validation
	UISchema                json.RawMessage `json:"uiSchema,omitempty"`                // UI Schema for rendering hints (optional)
	FormData                json.RawMessage `json:"formData,omitempty"`                // Default/pre-filled form data (optional)
	SubmissionURL           string          `json:"submissionUrl,omitempty"`           // URL to submit form data to (optional)
	RequiresOgaVerification bool            `json:"requiresOgaVerification,omitempty"` // If true, waits for OGA_VERIFICATION action; if false, completes after submission response
}

// SimpleFormResult represents the response data for form operations
type SimpleFormResult struct {
	FormID   string          `json:"formId,omitempty"`
	Title    string          `json:"title,omitempty"`
	Schema   json.RawMessage `json:"schema,omitempty"`
	UISchema json.RawMessage `json:"uiSchema,omitempty"`
	FormData json.RawMessage `json:"formData,omitempty"`
}

type SimpleForm struct {
	api         API
	config      Config
	cfg         *config.Config
	formService form.FormService
}

func NewSimpleForm(configJSON json.RawMessage, cfg *config.Config, formService form.FormService) (*SimpleForm, error) {

	var formConfig Config
	if err := json.Unmarshal(configJSON, &formConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &SimpleForm{
		config:      formConfig,
		cfg:         cfg,
		formService: formService,
	}, nil
}

func (s *SimpleForm) Init(api API) {
	s.api = api
}

func (s *SimpleForm) Start(ctx context.Context) (*ExecutionResponse, error) {
	// Populate form definition from registry if only formId is provided
	if s.config.FormID != "" && s.config.Schema == nil {
		if err := s.populateFromRegistry(ctx); err != nil {
			slog.Error("failed to populate form from registry", "formId", s.config.FormID, "error", err)
			return nil, fmt.Errorf("failed to populate form from registry: %w", err)
		}
	}

	// Initialize plugin state to DRAFT if not set
	if s.api.GetPluginState() == "" {
		if err := s.api.SetPluginState(string(TraderSavedAsDraft)); err != nil {
			slog.Error("failed to set initial plugin state", "error", err)
			return nil, fmt.Errorf("failed to set initial plugin state: %w", err)
		}
	}

	return &ExecutionResponse{
		Message: "SimpleForm task started successfully",
	}, nil
}

func (s *SimpleForm) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	// Default action is FETCH_FORM
	action := request.Action
	if action == "" {
		action = SimpleFormActionFetch
	}

	switch action {
	case SimpleFormActionFetch:
		return s.handleFetchForm(ctx)
	case SimpleFormActionSubmit:
		return s.handleSubmitForm(ctx, request.Content)
	case SimpleFormActionOgaVerify:
		return s.handleOgaVerification(ctx, request.Content)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// populateFromRegistry fills in the form definition from the registry based on formId
func (s *SimpleForm) populateFromRegistry(ctx context.Context) error {
	if s.formService == nil {
		return fmt.Errorf("form service is required to populate form definition")
	}

	// Parse form ID as UUID
	formUUID, err := uuid.Parse(s.config.FormID)
	if err != nil {
		return fmt.Errorf("invalid form ID format (expected UUID): %w", err)
	}

	// Get form from service
	def, err := s.formService.GetFormByID(ctx, formUUID)
	if err != nil {
		return fmt.Errorf("failed to get form definition for formId %s: %w", s.config.FormID, err)
	}

	s.config.Title = def.Name
	s.config.Schema = def.Schema
	s.config.UISchema = def.UISchema

	return nil
}

// handleFetchForm returns the form schema for rendering
func (s *SimpleForm) handleFetchForm(ctx context.Context) (*ExecutionResponse, error) {

	err := s.populateFromRegistry(ctx)

	if err != nil {
		return &ExecutionResponse{
			Message: fmt.Sprintf("Failed to populate form definition from registry: %v", err),
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "FETCH_FORM_FAILED",
					Message: "Failed to retrieve form definition.",
				},
			},
		}, err
	}
	// Prepopulate form data from global context
	prepopulatedFormData, err := s.prepopulateFormData(ctx, s.config.FormData)
	if err != nil {
		slog.Warn("failed to prepopulate form data from global context",
			"formId", s.config.FormID,
			"error", err)
		// Continue with original form data if prepopulation fails
		prepopulatedFormData = s.config.FormData
	}

	// Store prepopulated form data in local state for retrieval
	if err := s.api.WriteToLocalStore("prepopulatedFormData", prepopulatedFormData); err != nil {
		slog.Warn("failed to write prepopulated form data to local store", "error", err)
	}

	// Return the form schema (task stays in current state until form is submitted)
	return &ExecutionResponse{
		Message: "Form schema retrieved successfully",
		ApiResponse: &ApiResponse{
			Success: true,
			Data: SimpleFormResult{
				Title:    s.config.Title,
				Schema:   s.config.Schema,
				UISchema: s.config.UISchema,
				FormData: prepopulatedFormData,
			},
		},
	}, nil
}

// handleSubmitForm validates and processes the form submission
func (s *SimpleForm) handleSubmitForm(_ context.Context, content interface{}) (*ExecutionResponse, error) {
	// Parse form data from content
	formData, err := s.parseFormData(content)
	if err != nil {
		return &ExecutionResponse{
			Message: fmt.Sprintf("Invalid form data: %v", err),
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "INVALID_FORM_DATA",
					Message: "Invalid form Data, Parsing Failed.",
				},
			},
		}, err
	}

	// Convert formData to JSON for storage
	formDataJSON, err := json.Marshal(formData)
	if err != nil {
		return &ExecutionResponse{
			Message: fmt.Sprintf("Failed to process form data: %v", err),
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "INVALID_FORM_DATA",
					Message: "Failed to process form data.",
				},
			},
		}, err
	}

	// Store form data in local state
	if err := s.api.WriteToLocalStore("formData", formDataJSON); err != nil {
		slog.Warn("failed to write form data to local store", "error", err)
	}

	// Update plugin state to SUBMITTED
	if err := s.api.SetPluginState(string(TraderSubmitted)); err != nil {
		slog.Error("failed to set plugin state to SUBMITTED", "error", err)
	}

	// If submissionUrl is provided, send the form data to that URL
	if s.config.SubmissionURL != "" {
		requestPayload := map[string]any{
			"data":          formData,
			"taskId":        s.api.GetTaskID().String(),
			"consignmentId": s.api.GetConsignmentID().String(),
			"serviceUrl":    fmt.Sprintf("%s/api/tasks", s.cfg.Server.ServiceURL),
		}

		responseData, err := s.sendFormSubmission(s.config.SubmissionURL, requestPayload)
		if err != nil {
			slog.Error("failed to send form submission",
				"formId", s.config.FormID,
				"submissionUrl", s.config.SubmissionURL,
				"error", err)
			return &ExecutionResponse{
				Message: fmt.Sprintf("Failed to submit form to external system: %v", err),
				ApiResponse: &ApiResponse{
					Success: false,
					Error: &ApiError{
						Code:    "FORM_SUBMISSION_FAILED",
						Message: "Failed to submit form to external system.",
					},
				},
			}, err
		}

		// Check if OGA verification is required
		if s.config.RequiresOgaVerification {
			slog.Info("form submitted, waiting for OGA verification",
				"formId", s.config.FormID,
				"submissionUrl", s.config.SubmissionURL)

			newState := InProgress
			return &ExecutionResponse{
				NewState: &newState,
				Message:  "Form submitted successfully, awaiting OGA verification",
				ApiResponse: &ApiResponse{
					Success: true,
				},
			}, nil
		}

		// No OGA verification required - complete the task with response data
		slog.Info("form submitted and completed",
			"formId", s.config.FormID,
			"submissionUrl", s.config.SubmissionURL,
			"response", responseData)

		newState := Completed
		return &ExecutionResponse{
			NewState: &newState,
			Message:  "Form submitted and processed successfully",
			ApiResponse: &ApiResponse{
				Success: true,
			},
		}, nil
	}

	// If no submissionUrl, task is completed
	newState := Completed
	return &ExecutionResponse{
		NewState: &newState,
		Message:  "Form submitted successfully",
		ApiResponse: &ApiResponse{
			Success: true,
		},
	}, nil
}

// handleOgaVerification handles the OGA_VERIFICATION action and marks the task as completed
func (s *SimpleForm) handleOgaVerification(_ context.Context, content interface{}) (*ExecutionResponse, error) {
	verificationData, err := s.parseFormData(content)
	if err != nil {
		return &ExecutionResponse{
			Message: fmt.Sprintf("Invalid verification data: %v", err),
		}, err
	}

	// Check if verification was approved
	decision, ok := verificationData["decision"].(string)
	if !ok || strings.ToUpper(decision) != "APPROVED" {
		return &ExecutionResponse{
			Message: "Verification rejected or invalid",
		}, nil
	}

	// Update plugin state to OGA_REVIEWED
	if err := s.api.SetPluginState(string(OGAReviewed)); err != nil {
		slog.Error("failed to set plugin state to OGA_REVIEWED", "error", err)
	}

	// Mark task as completed
	newState := Completed
	return &ExecutionResponse{
		NewState: &newState,
		Message:  "Form verified by OGA, task completed",
	}, nil
}

// parseFormData parses the content into a map[string]interface{}
func (s *SimpleForm) parseFormData(content interface{}) (map[string]interface{}, error) {
	if content == nil {
		return nil, fmt.Errorf("content is required")
	}

	switch data := content.(type) {
	case map[string]interface{}:
		return data, nil
	default:
		// Try to marshal and unmarshal
		jsonBytes, err := json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal content: %w", err)
		}
		var formData map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &formData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content: %w", err)
		}
		return formData, nil
	}
}

// prepopulateFormData builds formData from schema by looking up values from global store
func (s *SimpleForm) prepopulateFormData(ctx context.Context, existingFormData json.RawMessage) (json.RawMessage, error) {
	// Parse the schema to build formData
	var schema map[string]interface{}
	if err := json.Unmarshal(s.config.Schema, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Build formData from schema
	formData := s.buildFormDataFromSchema(ctx, schema)

	// If we have existing formData, merge it (existing formData takes priority)
	if len(existingFormData) > 0 {
		var existingData map[string]interface{}
		if err := json.Unmarshal(existingFormData, &existingData); err == nil {
			formData = s.mergeFormData(formData, existingData)
		}
	}

	// If no data was populated, return existing
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
func (s *SimpleForm) buildFormDataFromSchema(ctx context.Context, schema map[string]interface{}) map[string]interface{} {
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
			if pathStr, ok := globalContextPath.(string); ok {
				if value := s.lookupValueFromGlobalStore(ctx, pathStr); value != nil {
					formData[fieldName] = value
				}
			}
		}

		// Handle nested objects recursively
		fieldType, _ := fieldDef["type"].(string)
		if fieldType == "object" {
			nestedData := s.buildFormDataFromSchema(ctx, fieldDef)
			if len(nestedData) > 0 {
				formData[fieldName] = nestedData
			}
		}
	}

	return formData
}

// lookupValueFromGlobalStore retrieves a value from global store using dot notation path
func (s *SimpleForm) lookupValueFromGlobalStore(ctx context.Context, path string) interface{} {
	if path == "" {
		return nil
	}

	// Split path by dots
	keys := splitPath(path)
	if len(keys) == 0 {
		return nil
	}

	// Read from global store
	value, found := s.api.ReadFromGlobalStore(keys[0])
	if !found {
		return nil
	}

	// Traverse nested keys
	current := value
	for i := 1; i < len(keys); i++ {
		if currentMap, ok := current.(map[string]interface{}); ok {
			var found bool
			current, found = currentMap[keys[i]]
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
func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ".")
}

// mergeFormData merges existing formData with prepopulated data
func (s *SimpleForm) mergeFormData(prepopulated, existing map[string]interface{}) map[string]interface{} {
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
				result[k] = s.mergeFormData(prepopMap, existingMap)
				continue
			}
		}
		// Otherwise, existing takes priority
		result[k] = v
	}

	return result
}

// sendFormSubmission sends the form data to the specified URL via HTTP POST
func (s *SimpleForm) sendFormSubmission(url string, formData map[string]interface{}) (map[string]interface{}, error) {
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
