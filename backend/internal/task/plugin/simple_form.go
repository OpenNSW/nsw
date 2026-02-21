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

// Resolved FSM actions for conditional transitions.
// The plugin's resolveAction method maps public API actions to these before FSM dispatch.
const (
	simpleFormFSMSubmitComplete = "SUBMIT_FORM_COMPLETE"
	simpleFormFSMSubmitAwaitOGA = "SUBMIT_FORM_AWAIT_OGA"
	simpleFormFSMOgaApproved    = "OGA_VERIFICATION_APPROVED"
	simpleFormFSMOgaRejected    = "OGA_VERIFICATION_REJECTED"
)

// SimpleFormState represents the current state the form is in
type SimpleFormState string

const (
	SimpleFormInitialized SimpleFormState = "INITIALIZED"
	TraderSavedAsDraft    SimpleFormState = "DRAFT"
	TraderSubmitted       SimpleFormState = "SUBMITTED"
	OGAAcknowledged       SimpleFormState = "OGA_ACKNOWLEDGED"
	OGAReviewed           SimpleFormState = "OGA_REVIEWED"
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

type Meta struct {
	VerificationType string `json:"type"`
	VerificationId   string `json:"verificationId"`
}
type Request struct {
	Meta *Meta `json:"meta"`
}
type Response struct {
	Mapping map[string]string `json:"mapping,omitempty"` // Data to be mapped to global context after submission
	Display *Display          `json:"display,omitempty"`
}

type Display struct {
	FormID string `json:"formId"`
}

type SubmissionConfig struct {
	Url      string    `json:"url"` // URL to submit form data to
	Request  *Request  `json:"request,omitempty"`
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

// NewSimpleFormFSM returns the state graph for SimpleForm.
// It lives here, next to the plugin that owns it.
//
// State graph:
//
//	""               ──START──────────────────────► INITIALIZED      [no task state change]
//	INITIALIZED      ──DRAFT_FORM─────────────────► DRAFT            [IN_PROGRESS]
//	INITIALIZED      ──SUBMIT_FORM_COMPLETE────────► SUBMITTED        [COMPLETED]
//	INITIALIZED      ──SUBMIT_FORM_AWAIT_OGA───────► OGA_ACKNOWLEDGED [IN_PROGRESS]
//	DRAFT            ──DRAFT_FORM─────────────────► DRAFT            [IN_PROGRESS]
//	DRAFT            ──SUBMIT_FORM_COMPLETE────────► SUBMITTED        [COMPLETED]
//	DRAFT            ──SUBMIT_FORM_AWAIT_OGA───────► OGA_ACKNOWLEDGED [IN_PROGRESS]
//	OGA_ACKNOWLEDGED ──OGA_VERIFICATION_APPROVED──► OGA_REVIEWED     [COMPLETED]
//	OGA_ACKNOWLEDGED ──OGA_VERIFICATION_REJECTED──► OGA_REVIEWED     [FAILED]
func NewSimpleFormFSM() *PluginFSM {
	return NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}: {string(SimpleFormInitialized), ""},

		{string(SimpleFormInitialized), SimpleFormActionDraft}: {string(TraderSavedAsDraft), InProgress},
		{string(TraderSavedAsDraft), SimpleFormActionDraft}:    {string(TraderSavedAsDraft), InProgress},

		{string(SimpleFormInitialized), simpleFormFSMSubmitComplete}: {string(TraderSubmitted), Completed},
		{string(TraderSavedAsDraft), simpleFormFSMSubmitComplete}:    {string(TraderSubmitted), Completed},

		{string(SimpleFormInitialized), simpleFormFSMSubmitAwaitOGA}: {string(OGAAcknowledged), InProgress},
		{string(TraderSavedAsDraft), simpleFormFSMSubmitAwaitOGA}:    {string(OGAAcknowledged), InProgress},

		{string(OGAAcknowledged), simpleFormFSMOgaApproved}: {string(OGAReviewed), Completed},
		{string(OGAAcknowledged), simpleFormFSMOgaRejected}: {string(OGAReviewed), Failed},
	})
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

func (s *SimpleForm) GetRenderInfo(ctx context.Context) (*ApiResponse, error) {
	if err := s.populateFromRegistry(ctx); err != nil {
		return &ApiResponse{
			Success: false,
			Error: &ApiError{
				Code:    "FETCH_FORM_FAILED",
				Message: "Failed to retrieve form definition.",
			},
		}, err
	}

	pluginState := SimpleFormState(s.api.GetPluginState())

	formData, err := s.resolveFormData(ctx, pluginState)
	if err != nil {
		slog.Warn("failed to resolve form data, falling back to defaults",
			"formId", s.config.FormID, "pluginState", pluginState, "error", err)
		formData = s.config.FormData
	}

	content := map[string]any{
		"traderFormInfo": map[string]any{
			"title":    s.config.Title,
			"uiSchema": s.config.UISchema,
			"formData": formData,
			"schema":   s.config.Schema,
		},
	}

	if s.config.Submission != nil {
		s.attachFormDisplay(ctx, content, "submissionResponse", displayFormID(s.config.Submission.Response), "submissionResponseForm")
	}
	if s.config.Callback != nil {
		s.attachFormDisplay(ctx, content, "ogaResponse", displayFormID(s.config.Callback.Response), "ogaReviewForm")
	}

	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypeSimpleForm,
			State:       s.api.GetTaskState(),
			PluginState: string(pluginState),
			Content:     content,
		},
	}, nil
}

func (s *SimpleForm) Start(ctx context.Context) (*ExecutionResponse, error) {
	if !s.api.CanTransition(FSMActionStart) {
		return &ExecutionResponse{Message: "SimpleForm task already started"}, nil
	}
	if s.config.FormID != "" && s.config.Schema == nil {
		if err := s.populateFromRegistry(ctx); err != nil {
			slog.Error("failed to populate form from registry", "formId", s.config.FormID, "error", err)
			return nil, fmt.Errorf("failed to populate form from registry: %w", err)
		}
	}
	if err := s.api.Transition(FSMActionStart); err != nil {
		return nil, err
	}
	return &ExecutionResponse{Message: "SimpleForm task started successfully"}, nil
}

func (s *SimpleForm) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	action, err := s.resolveAction(request)
	if err != nil {
		return nil, err
	}
	if !s.api.CanTransition(action) {
		return nil, fmt.Errorf("fsm: action %q not permitted in state %q", request.Action, s.api.GetPluginState())
	}
	resp, err := s.dispatch(ctx, action, request.Content)
	if err != nil {
		return resp, err
	}
	if err := s.api.Transition(action); err != nil {
		return nil, err
	}
	return resp, nil
}

// resolveAction maps the public API action string to an FSM action.
// For actions with conditional outcomes it inspects the request to pick the right edge.
func (s *SimpleForm) resolveAction(request *ExecutionRequest) (string, error) {
	switch request.Action {
	case SimpleFormActionSubmit:
		if s.requiresVerification() {
			return simpleFormFSMSubmitAwaitOGA, nil
		}
		return simpleFormFSMSubmitComplete, nil

	case SimpleFormActionOgaVerify:
		data, err := s.parseFormData(request.Content)
		if err != nil {
			return "", fmt.Errorf("invalid verification data: %w", err)
		}
		decision, _ := data["decision"].(string)
		if strings.ToUpper(decision) == "APPROVED" { // TODO: Need to change this hardcoded APPROVED
			return simpleFormFSMOgaApproved, nil
		}
		return simpleFormFSMOgaRejected, nil

	default:
		return request.Action, nil
	}
}

// dispatch routes an already-resolved FSM action to the appropriate handler.
func (s *SimpleForm) dispatch(ctx context.Context, action string, content any) (*ExecutionResponse, error) {
	switch action {
	case SimpleFormActionDraft:
		return s.draftHandler(ctx, content)
	case simpleFormFSMSubmitComplete, simpleFormFSMSubmitAwaitOGA:
		return s.submitHandler(ctx, content)
	case simpleFormFSMOgaApproved:
		return s.ogaApprovedHandler(ctx, content)
	case simpleFormFSMOgaRejected:
		return s.ogaRejectedHandler(ctx, content)
	default:
		return nil, fmt.Errorf("unhandled FSM action: %q", action)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// draftHandler saves the current form data as a draft to local store.
func (s *SimpleForm) draftHandler(_ context.Context, content any) (*ExecutionResponse, error) {
	if err := s.api.WriteToLocalStore("trader:form", content); err != nil {
		return &ExecutionResponse{
			ApiResponse: &ApiResponse{
				Success: false,
				Error:   &ApiError{Code: "SAVE_DRAFT_FAILED", Message: "Failed to save draft."},
			},
		}, err
	}
	return &ExecutionResponse{ApiResponse: &ApiResponse{Success: true}}, nil
}

// submitHandler is shared by SUBMIT_FORM_COMPLETE and SUBMIT_FORM_AWAIT_OGA.
// It persists the submission, extracts global context values, and optionally
// sends the data to an external service.
func (s *SimpleForm) submitHandler(ctx context.Context, content any) (*ExecutionResponse, error) {
	formData, err := s.parseFormData(content)
	if err != nil {
		return &ExecutionResponse{
			ApiResponse: &ApiResponse{
				Success: false,
				Error:   &ApiError{Code: "INVALID_FORM_DATA", Message: "Invalid form Data, Parsing Failed."},
			},
		}, err
	}

	if err := s.api.WriteToLocalStore("trader:form", formData); err != nil {
		slog.Warn("failed to write form data to local store", "error", err)
	}

	if err := s.populateFromRegistry(ctx); err != nil {
		return &ExecutionResponse{
			ApiResponse: &ApiResponse{
				Success: false,
				Error:   &ApiError{Code: "INVALID_FORM_DATA", Message: "Failed to process form data."},
			},
		}, err
	}

	var parsedSchema jsonform.JSONSchema
	if err := json.Unmarshal(s.config.Schema, &parsedSchema); err != nil {
		return &ExecutionResponse{
			ApiResponse: &ApiResponse{
				Success: false,
				Error:   &ApiError{Code: "INVALID_FORM_DATA", Message: "Failed to parse schema."},
			},
		}, err
	}

	globalContextPairs := make(map[string]any)
	err = jsonform.Traverse(&parsedSchema, func(path string, node *jsonform.JSONSchema, parent *jsonform.JSONSchema) error {
		if node.Type == "string" || node.Type == "number" || node.Type == "boolean" {
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
			ApiResponse: &ApiResponse{
				Success: false,
				Error:   &ApiError{Code: "INVALID_FORM_DATA", Message: "Failed to process form data."},
			},
		}, err
	}

	submissionUrl := s.submissionUrl()
	if submissionUrl == "" {
		return &ExecutionResponse{
			AppendGlobalContext: globalContextPairs,
			Message:             "Form submitted successfully",
			ApiResponse:         &ApiResponse{Success: true},
		}, nil
	}

	requestPayload := map[string]any{
		"data":       formData,
		"taskId":     s.api.GetTaskID().String(),
		"workflowId": s.api.GetWorkflowID().String(),
		"serviceUrl": strings.TrimRight(s.cfg.Server.ServiceURL, "/") + TasksAPIPath,
	}
	if s.config.Submission != nil && s.config.Submission.Request != nil {
		requestPayload["meta"] = s.config.Submission.Request.Meta
	}

	responseData, err := s.sendFormSubmission(submissionUrl, requestPayload)
	if err != nil {
		slog.Error("failed to send form submission",
			"formId", s.config.FormID, "submissionUrl", submissionUrl, "error", err)
		return &ExecutionResponse{
			AppendGlobalContext: globalContextPairs,
			ApiResponse: &ApiResponse{
				Success: false,
				Error:   &ApiError{Code: "FORM_SUBMISSION_FAILED", Message: "Failed to submit form to external system."},
			},
		}, err
	}

	if err := s.api.WriteToLocalStore("submissionResponse", responseData); err != nil {
		slog.Warn("failed to write submission response to local store", "formId", s.config.FormID, "error", err)
	}

	if s.config.Submission != nil &&
		s.config.Submission.Response != nil &&
		s.config.Submission.Response.Mapping != nil {
		slog.Info("received response from form submission, parsing based on expected mapping",
			"formId", s.config.FormID, "submissionUrl", submissionUrl, "response", responseData)
		parsed, err := s.parseResponseData(responseData, s.config.Submission.Response.Mapping)
		if err != nil {
			slog.Warn("failed to parse some submission response data fields, continuing with what was found",
				"formId", s.config.FormID, "submissionUrl", submissionUrl, "error", err)
		}
		for k, v := range parsed {
			globalContextPairs[k] = v
		}
	}

	return &ExecutionResponse{
		AppendGlobalContext: globalContextPairs,
		Message:             "Form submitted successfully",
		ApiResponse:         &ApiResponse{Success: true},
	}, nil
}

// ogaApprovedHandler handles OGA_VERIFICATION_APPROVED: stores the OGA response
// and maps callback fields to global context.
func (s *SimpleForm) ogaApprovedHandler(_ context.Context, content any) (*ExecutionResponse, error) {
	verificationData, err := s.parseAndStoreOgaResponse(content)
	if err != nil {
		return nil, err
	}

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
				"formId", s.config.FormID, "error", err)
		}
		for k, v := range parsed {
			globalContextPairs[k] = v
		}
	}

	return &ExecutionResponse{
		AppendGlobalContext: globalContextPairs,
		Message:             "Form verified by OGA, task completed",
	}, nil
}

// ogaRejectedHandler handles OGA_VERIFICATION_REJECTED: stores the rejection response.
func (s *SimpleForm) ogaRejectedHandler(_ context.Context, content any) (*ExecutionResponse, error) {
	if _, err := s.parseAndStoreOgaResponse(content); err != nil {
		return nil, err
	}
	return &ExecutionResponse{Message: "Verification rejected or invalid"}, nil
}

// parseAndStoreOgaResponse parses the OGA payload and persists it to local store.
func (s *SimpleForm) parseAndStoreOgaResponse(content any) (map[string]any, error) {
	verificationData, err := s.parseFormData(content)
	if err != nil {
		return nil, fmt.Errorf("invalid verification data: %w", err)
	}
	if err := s.api.WriteToLocalStore("ogaResponse", verificationData); err != nil {
		return nil, err
	}
	return verificationData, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// resolveFormData returns the appropriate form data for the current plugin state.
func (s *SimpleForm) resolveFormData(ctx context.Context, state SimpleFormState) (any, error) {
	switch state {
	case SimpleFormInitialized:
		return s.prepopulateFormData(ctx, s.config.FormData)
	case TraderSavedAsDraft, TraderSubmitted, OGAAcknowledged, OGAReviewed:
		return s.api.ReadFromLocalStore("trader:form")
	default:
		return s.config.FormData, nil
	}
}

// displayFormID extracts the display form ID from a Response config, returning "" if unset.
func displayFormID(r *Response) string {
	if r != nil && r.Display != nil {
		return r.Display.FormID
	}
	return ""
}

// attachFormDisplay fetches a form definition and attaches it to content under contentKey,
// using data read from storeKey as the formData. A no-op if formID or stored data is absent.
func (s *SimpleForm) attachFormDisplay(ctx context.Context, content map[string]any, storeKey, formID, contentKey string) {
	if formID == "" {
		return
	}
	data, err := s.api.ReadFromLocalStore(storeKey)
	if err != nil || data == nil {
		if err != nil {
			slog.Warn("failed to read from local store", "formId", s.config.FormID, "key", storeKey, "error", err)
		}
		return
	}
	id, err := uuid.Parse(formID)
	if err != nil {
		slog.Warn("invalid display form ID, expected UUID", "formId", s.config.FormID, "displayFormId", formID, "error", err)
		return
	}
	def, err := s.formService.GetFormByID(ctx, id)
	if err != nil {
		slog.Warn("failed to fetch display form definition", "formId", s.config.FormID, "displayFormId", formID, "error", err)
		return
	}
	content[contentKey] = map[string]any{
		"title":    def.Name,
		"uiSchema": def.UISchema,
		"schema":   def.Schema,
		"formData": data,
	}
}

func (s *SimpleForm) populateFromRegistry(ctx context.Context) error {
	if s.formService == nil {
		return fmt.Errorf("form service is required to populate form definition")
	}
	formUUID, err := uuid.Parse(s.config.FormID)
	if err != nil {
		return fmt.Errorf("invalid form ID format (expected UUID): %w", err)
	}
	def, err := s.formService.GetFormByID(ctx, formUUID)
	if err != nil {
		return fmt.Errorf("failed to get form definition for formId %s: %w", s.config.FormID, err)
	}
	s.config.Title = def.Name
	s.config.Schema = def.Schema
	s.config.UISchema = def.UISchema
	return nil
}

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
	client := &http.Client{Timeout: 30 * time.Second}
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
			slog.Warn("failed to parse response as JSON, storing as raw string", "url", url, "error", err)
			responseData = map[string]interface{}{"raw_response": string(body)}
		}
	}
	slog.Info("form submitted successfully", "url", url, "status", resp.StatusCode, "response", responseData)
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
