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
	"github.com/OpenNSW/nsw/pkg/jsonform"
)

// SimpleFormAction represents the action to perform on the form
const (
	SimpleFormActionDraft     = "DRAFT_FORM"
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

const TasksAPIPath = "/api/v1/tasks"

// Config contains the JSON Form configuration
type Config struct {
	FormID                  string            `json:"formId"`                  // Unique identifier for the form
	Title                   string            `json:"title"`                   // Display title of the form
	Schema                  json.RawMessage   `json:"schema"`                  // JSON Schema defining the form structure and validation
	UISchema                json.RawMessage   `json:"uiSchema,omitempty"`      // UI Schema for rendering hints (optional)
	FormData                json.RawMessage   `json:"formData,omitempty"`      // Default/pre-filled form data (optional)
	SubmissionURL           string            `json:"submissionUrl,omitempty"` // URL to submit form data to (optional)
	Submission              *SubmissionConfig `json:"submission,omitempty"`    // Submission configuration (optional)
	Callback                *CallbackConfig   `json:"callback,omitempty"`
	RequiresOgaVerification bool              `json:"requiresOgaVerification,omitempty"` // If true, waits for OGA_VERIFICATION action; if false, completes after submission response
}

type Response struct {
	Mapping map[string]string `json:"mapping,omitempty"` // Data to be mapped to global context after submission
}

type SubmissionConfig struct {
	Url      string    `json:"url"`                // URL to submit form data to
	Response *Response `json:"response,omitempty"` // Expected response mapping after submission
}

type CallbackConfig struct {
	Response *Response `json:"response,omitempty"`
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

func (s *SimpleForm) GetRenderInfo(ctx context.Context) (*ApiResponse, error) {

	err := s.populateFromRegistry(ctx)

	if err != nil {
		return &ApiResponse{
			Success: false,
			Error: &ApiError{
				Code:    "FETCH_FORM_FAILED",
				Message: "Failed to retrieve form definition.",
			},
		}, err
	}

	pluginState := s.api.GetPluginState()
	ogaResponse, err := s.api.ReadFromLocalStore("ogaResponse")

	if err != nil {
		slog.Warn("failed to read ogaResponse from local store", "error", err)
	}
	var prepopulatedFormData any
	// Prepopulate form data from global context
	switch pluginState {
	case string(Initialized):
		prepopulatedFormData, err = s.prepopulateFormData(ctx, s.config.FormData)
	case string(TraderSavedAsDraft):
		prepopulatedFormData, err = s.api.ReadFromLocalStore("trader:draft")
	case string(TraderSubmitted), string(OGAAcknowledged), string(OGAReviewed):
		prepopulatedFormData, err = s.api.ReadFromLocalStore("trader:submission")
	default:
		prepopulatedFormData = s.config.FormData
	}

	if err != nil {
		slog.Warn("failed to prepopulate form data from global context",
			"formId", s.config.FormID,
			"error", err)
		// Continue with original form data if prepopulation fails
		prepopulatedFormData = s.config.FormData
	}

	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypeSimpleForm,
			State:       s.api.GetTaskState(),
			PluginState: pluginState,
			Content: map[string]any{
				"traderFormInfo": map[string]any{
					"title":    s.config.Title,
					"uiSchema": s.config.UISchema,
					"formData": prepopulatedFormData,
					"schema":   s.config.Schema,
				},
				"ogaResponse": ogaResponse,
			},
		},
	}, nil
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

	// Set initial plugin state if not already set
	if s.api.GetPluginState() == "" {
		if err := s.api.SetPluginState(string(Initialized)); err != nil {
			slog.Error("failed to set initial plugin state", "error", err)
			return nil, fmt.Errorf("failed to set initial plugin state: %w", err)
		}
	}

	return &ExecutionResponse{
		Message: "SimpleForm task started successfully",
	}, nil
}

func (s *SimpleForm) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	action := request.Action

	switch action {
	case SimpleFormActionDraft:
		return s.handleSaveAsDraft(ctx, request.Content)
	case SimpleFormActionSubmit:
		return s.handleSubmitForm(ctx, request.Content)
	case SimpleFormActionOgaVerify:
		return s.handleOgaVerification(ctx, request.Content)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (s *SimpleForm) handleSaveAsDraft(_ context.Context, content any) (*ExecutionResponse, error) {
	err := s.api.WriteToLocalStore("trader:draft", content)
	if err != nil {
		return &ExecutionResponse{
			Message: fmt.Sprintf("Failed to save draft: %v", err),
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "SAVE_DRAFT_FAILED",
					Message: "Failed to save draft.",
				},
			},
		}, err
	}

	pluginState := string(TraderSavedAsDraft)
	if err := s.api.SetPluginState(pluginState); err != nil {
		slog.Error("failed to set plugin state to SAVED_AS_DRAFT", "error", err)
	}

	newState := InProgress

	return &ExecutionResponse{
		NewState:      &newState,
		ExtendedState: &pluginState,
		ApiResponse: &ApiResponse{
			Success: true,
		},
	}, nil
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

// handleSubmitForm validates and processes the form submission
func (s *SimpleForm) handleSubmitForm(ctx context.Context, content interface{}) (*ExecutionResponse, error) {
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

	// Store form data in local state
	if err := s.api.WriteToLocalStore("trader:submission", formData); err != nil {
		slog.Warn("failed to write form data to local store", "error", err)
	}

	if err := s.populateFromRegistry(ctx); err != nil {
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

	// Convert the schema to JSONSchema
	var parsedSchema jsonform.JSONSchema

	if err := json.Unmarshal(s.config.Schema, &parsedSchema); err != nil {
		return &ExecutionResponse{
			Message: fmt.Sprintf("Failed to parse schema: %v", err),
			ApiResponse: &ApiResponse{
				Success: false,
				Error: &ApiError{
					Code:    "INVALID_FORM_DATA",
					Message: "Failed to parse schema.",
				},
			},
		}, err
	}

	globalContextPairs := make(map[string]any)

	// Traverse through formData and check for globalContext paths
	err = jsonform.Traverse(&parsedSchema, func(path string, node *jsonform.JSONSchema, parent *jsonform.JSONSchema) error {
		if node.Type == "string" || node.Type == "number" || node.Type == "boolean" {
			// Get the value from formData using the path

			if node.XGlobalContext != nil &&
				node.XGlobalContext.WriteTo != nil &&
				strings.TrimSpace(*node.XGlobalContext.WriteTo) != "" {

				value, exists := jsonform.GetValueByPath(formData, path)
				if !exists {
					return fmt.Errorf("value for global context path '%s' not found in submitted form data", *node.XGlobalContext.WriteTo)
				}

				globalContextPairs[*node.XGlobalContext.WriteTo] = value
			}
		}
		return nil
	})

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

	// Update plugin state to TRADER_SUBMITTED
	if err := s.api.SetPluginState(string(TraderSubmitted)); err != nil {
		slog.Error("failed to set plugin state to SUBMITTED", "error", err)
	}
	pluginState := string(TraderSubmitted)

	// If submissionUrl is provided, send the form data to that URL
	submissionUrl := s.submissionUrl()
	if submissionUrl != "" {
		requestPayload := map[string]any{
			"data":       formData,
			"taskId":     s.api.GetTaskID().String(),
			"workflowId": s.api.GetWorkflowID().String(),
			"serviceUrl": strings.TrimRight(s.cfg.Server.ServiceURL, "/") + TasksAPIPath,
		}

		responseData, err := s.sendFormSubmission(submissionUrl, requestPayload)
		if err != nil {
			slog.Error("failed to send form submission",
				"formId", s.config.FormID,
				"submissionUrl", submissionUrl,
				"error", err)
			return &ExecutionResponse{
				Message:             fmt.Sprintf("Failed to submit form to external system: %v", err),
				AppendGlobalContext: globalContextPairs,
				ApiResponse: &ApiResponse{
					Success: false,
					Error: &ApiError{
						Code:    "FORM_SUBMISSION_FAILED",
						Message: "Failed to submit form to external system.",
					},
				},
			}, err
		}

		// If there are expected response mappings, to global state keys, parse the response accordingly and store in
		// global context for future use (e.g. in OGA verification or downstream tasks)
		if s.config.Submission != nil &&
			s.config.Submission.Response != nil &&
			s.config.Submission.Response.Mapping != nil {
			slog.Info("received response from form submission, parsing based on expected mapping",
				"formId", s.config.FormID,
				"submissionUrl", submissionUrl,
				"response", responseData)

			parsed, err := s.parseResponseData(responseData, s.config.Submission.Response.Mapping)
			if err != nil {
				slog.Warn("failed to parse some submission response data fields, continuing with what was found",
					"formId", s.config.FormID,
					"submissionUrl", submissionUrl,
					"error", err)
			}
			if len(parsed) > 0 {
				for k, v := range parsed {
					globalContextPairs[k] = v
				}
			}
		}

		// Check if OGA verification is required
		if s.requiresVerification() {
			slog.Info("form submitted, waiting for OGA verification",
				"formId", s.config.FormID,
				"submissionUrl", submissionUrl)

			// Update plugin state to OGA_ACKNOWLEDGED
			if err := s.api.SetPluginState(string(OGAAcknowledged)); err != nil {
				slog.Error("failed to set plugin state to OGA_ACKNOWLEDGED", "error", err)
			}
			pluginState = string(OGAAcknowledged)

			newState := InProgress
			return &ExecutionResponse{
				AppendGlobalContext: globalContextPairs,
				NewState:            &newState,
				ExtendedState:       &pluginState,
				Message:             "Form submitted successfully, awaiting OGA verification",
				ApiResponse: &ApiResponse{
					Success: true,
				},
			}, nil
		}

		// No OGA verification required - complete the task with response data
		slog.Info("form submitted and completed",
			"formId", s.config.FormID,
			"submissionUrl", submissionUrl,
			"response", responseData)

		newState := Completed
		return &ExecutionResponse{
			AppendGlobalContext: globalContextPairs,
			NewState:            &newState,
			ExtendedState:       &pluginState,
			Message:             "Form submitted and processed successfully",
			ApiResponse: &ApiResponse{
				Success: true,
			},
		}, nil
	}

	// If no submissionUrl, task is completed
	newState := Completed
	return &ExecutionResponse{
		AppendGlobalContext: globalContextPairs,
		NewState:            &newState,
		ExtendedState:       &pluginState,
		Message:             "Form submitted successfully",
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

	err = s.api.WriteToLocalStore("ogaResponse", verificationData)

	if err != nil {
		return nil, err
	}

	// Update plugin state to OGA_REVIEWED (common for both approved and rejected)
	if err := s.api.SetPluginState(string(OGAReviewed)); err != nil {
		slog.Error("failed to set plugin state to OGA_REVIEWED", "error", err)
	}
	pluginState := string(OGAReviewed)

	// Check if verification was approved
	decision, ok := verificationData["decision"].(string)
	if !ok || strings.ToUpper(decision) != "APPROVED" { // TODO: Need to change this hardcoded APPROVED
		// Mark task as FAILED
		newState := Failed
		return &ExecutionResponse{
			NewState:      &newState,
			ExtendedState: &pluginState,
			Message:       "Verification rejected or invalid",
		}, nil
	}

	// If there are any mapping of OGA verification response to global context, parse and store in global context for future use(e.g. downstream tasks)
	globalContextPairs := make(map[string]any)

	if s.config.Callback != nil &&
		s.config.Callback.Response != nil &&
		s.config.Callback.Response.Mapping != nil {
		slog.Info("parsing OGA verification response based on expected mapping",
			"formId", s.config.FormID,
			"mapping", s.config.Callback.Response.Mapping,
			"verificationData", verificationData)

		parsed, err := s.parseResponseData(verificationData, s.config.Callback.Response.Mapping)

		if err != nil {
			slog.Warn("failed to parse some OGA verification response data fields, continuing with what was found",
				"formId", s.config.FormID,
				"error", err)
		}
		if len(parsed) > 0 {
			for k, v := range parsed {
				globalContextPairs[k] = v
			}
		}
	}

	// Mark task as COMPLETED
	newState := Completed
	return &ExecutionResponse{
		NewState:            &newState,
		ExtendedState:       &pluginState,
		AppendGlobalContext: globalContextPairs,
		Message:             "Form verified by OGA, task completed",
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
	// Parse the schema using JSONSchema struct
	var parsedSchema jsonform.JSONSchema
	if err := json.Unmarshal(s.config.Schema, &parsedSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Build formData from schema using traverse
	formData := make(map[string]any)

	err := jsonform.Traverse(&parsedSchema, func(path string, node *jsonform.JSONSchema, parent *jsonform.JSONSchema) error {
		// Check if this field should be read from a global context
		if node.XGlobalContext != nil &&
			node.XGlobalContext.ReadFrom != nil &&
			strings.TrimSpace(*node.XGlobalContext.ReadFrom) != "" {

			// Lookup value from global store
			value := s.lookupValueFromGlobalStore(ctx, *node.XGlobalContext.ReadFrom)
			if value != nil {
				// Set the value at the current path in formData
				jsonform.SetValueByPath(formData, path, value)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to traverse schema for prepopulation: %w", err)
	}

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

// lookupValueFromGlobalStore retrieves a value from global store using dot notation path
func (s *SimpleForm) lookupValueFromGlobalStore(_ context.Context, path string) interface{} {
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

// submissionUrl returns the submission URL, preferring Submission.Url over the deprecated SubmissionURL.
func (s *SimpleForm) submissionUrl() string {
	if s.config.Submission != nil && s.config.Submission.Url != "" {
		return s.config.Submission.Url
	}
	return s.config.SubmissionURL
}

// requiresVerification checks if callback configuration is provided
func (s *SimpleForm) requiresVerification() bool {
	return s.config.RequiresOgaVerification || s.config.Callback != nil
}

// parseResponseData is a helper function to parse response data based on expected mapping.
// It returns all successfully mapped values, and an error containing a list of fields that were not found.
func (s *SimpleForm) parseResponseData(responseData map[string]any, mapping map[string]string) (map[string]any, error) {
	parsedData := make(map[string]any)
	var missingFields []string
	for k, v := range mapping {
		value, exists := jsonform.GetValueByPath(responseData, k)

		if !exists {
			missingFields = append(missingFields, k)
			continue
		}

		parsedData[v] = value
	}

	if len(missingFields) > 0 {
		return parsedData, fmt.Errorf("expected response field(s) not found: %s", strings.Join(missingFields, ", "))
	}

	return parsedData, nil
}
