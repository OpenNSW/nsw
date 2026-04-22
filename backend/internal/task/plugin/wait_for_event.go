package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/pkg/jsonform"
	"github.com/OpenNSW/nsw/pkg/jsonutils"
	"github.com/OpenNSW/nsw/pkg/remote"
)

type waitForEventState string

const (
	notifiedService  waitForEventState = "NOTIFIED_SERVICE"
	notifyFailed     waitForEventState = "NOTIFY_FAILED"
	receivedCallback waitForEventState = "RECEIVED_CALLBACK"
)

type DisplayState string

const (
	DisplayStateWaiting   DisplayState = "waiting"
	DisplayStateFailed    DisplayState = "failed"
	DisplayStateCompleted DisplayState = "completed"
)

// Internal FSM actions for WaitForEventTask.
const (
	waitForEventFSMStartFailed = "START_FAILED"
	waitForEventFSMRetry       = "RETRY"
	waitForEventFSMComplete    = "OGA_VERIFICATION"
)

// WaitForEventDisplay holds optional UI display metadata for the portal
type WaitForEventDisplay struct {
	Title       any `json:"title"`
	Description any `json:"description"`
}

// If Title or Description in WaitForEventDisplay is an object, it should have the following structure to support different text based on task state.
// Else it will be treated as static text.
type WaitForEventExtendedDisplay struct {
	Waiting   *string `json:"waiting,omitempty"`
	Failed    *string `json:"failed,omitempty"`
	Completed *string `json:"completed,omitempty"`
}

// WaitForEventConfig represents the configuration for a WAIT_FOR_EVENT task
type WaitForEventConfig struct {
	Display    *WaitForEventDisplay `json:"display,omitempty"`
	Submission *SubmissionConfig    `json:"submission,omitempty"`
}

type WaitForEventTask struct {
	api            API
	config         WaitForEventConfig
	serviceBaseURL string
	remoteManager  *remote.Manager
	formService    form.FormService
}

func (t *WaitForEventTask) GetRenderInfo(ctx context.Context) (*ApiResponse, error) {
	return &ApiResponse{
		Success: true,
		Data: GetRenderInfoResponse{
			Type:        TaskTypeWaitForEvent,
			PluginState: t.api.GetPluginState(),
			State:       t.api.GetTaskState(),
			Content:     t.renderContent(ctx),
		},
	}, nil
}

func (t *WaitForEventTask) Init(api API) {
	t.api = api
}

// WaitForEventExternalServiceRequest represents the payload sent to the external service
type WaitForEventExternalServiceRequest struct {
	TaskCode   string `json:"taskCode"` // Code to identify task config on External service side
	TaskID     string `json:"taskId"`
	WorkflowID string `json:"workflowId"`
	ServiceURL string `json:"serviceUrl"`
	Data       any    `json:"data"` // Resolved data from global context
}

// NewWaitForEventFSM returns the state graph for WaitForEventTask.
//
// State graph:
//
//	""               ──START────────► NOTIFIED_SERVICE  [IN_PROGRESS]
//	""               ──START_FAILED─► NOTIFY_FAILED     [IN_PROGRESS]
//	NOTIFY_FAILED    ──RETRY────────► NOTIFIED_SERVICE  [IN_PROGRESS]
//	NOTIFIED_SERVICE ──COMPLETE─────► RECEIVED_CALLBACK [COMPLETED]
func NewWaitForEventFSM() *PluginFSM {
	return NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}:                               {string(notifiedService), InProgress},
		{"", waitForEventFSMStartFailed}:                   {string(notifyFailed), InProgress},
		{string(notifyFailed), waitForEventFSMRetry}:       {string(notifiedService), InProgress},
		{string(notifiedService), waitForEventFSMComplete}: {string(receivedCallback), Completed},
	})
}

func (t *WaitForEventTask) renderContent(ctx context.Context) map[string]any {
	content := map[string]any{}
	var err error
	if t.config.Display != nil {
		state := waitForEventState(t.api.GetPluginState())
		content["display"], err = t.getDisplay(state)
		if err != nil {
			slog.Warn("failed to get display for wait_for_event task, using empty display", "taskId", t.api.GetTaskID(), "error", err)
			content["display"] = &WaitForEventDisplay{}
		}
	}
	// Attach OGA/Reviewer response if it exists in local store
	if t.config.Submission != nil && t.config.Submission.Response != nil {
		t.attachFormDisplay(ctx, content, "eventResponse", displayFormID(t.config.Submission.Response), "eventReviewForm")
	}

	return content
}

func NewWaitForEventTask(raw json.RawMessage, serviceBaseURL string, remoteManager *remote.Manager, formService form.FormService) (*WaitForEventTask, error) {
	var cfg WaitForEventConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &WaitForEventTask{
		config:         cfg,
		serviceBaseURL: serviceBaseURL,
		remoteManager:  remoteManager,
		formService:    formService,
	}, nil
}

func (t *WaitForEventTask) Start(ctx context.Context) (*ExecutionResponse, error) {
	if !t.api.CanTransition(FSMActionStart) {
		return &ExecutionResponse{Message: "WaitForEvent already started"}, nil
	}
	if t.config.Submission == nil || t.config.Submission.Url == "" {
		return nil, fmt.Errorf("submission url not configured in task config")
	}
	if err := t.notifyExternalService(ctx, t.api.GetTaskID(), t.api.GetWorkflowID()); err != nil {
		if transErr := t.api.Transition(waitForEventFSMStartFailed); transErr != nil {
			slog.ErrorContext(ctx, "failed to transition to NOTIFY_FAILED after notification error",
				"taskId", t.api.GetTaskID(),
				"workflowId", t.api.GetWorkflowID(),
				"error", transErr)
			return nil, fmt.Errorf("failed to notify external service and transition to NOTIFY_FAILED: %w", transErr)
		}
		return &ExecutionResponse{Message: "Failed to notify external service"}, nil
	}
	if err := t.api.Transition(FSMActionStart); err != nil {
		return nil, err
	}
	return &ExecutionResponse{Message: "Notified external service, waiting for callback"}, nil
}

func (t *WaitForEventTask) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("execution request is required")
	}
	switch request.Action {
	case waitForEventFSMRetry:
		if err := t.notifyExternalService(ctx, t.api.GetTaskID(), t.api.GetWorkflowID()); err != nil {
			// error is nil since the problem is not on system side.
			return &ExecutionResponse{
				Message: "Failed to notify external service",
				ApiResponse: &ApiResponse{
					Success: false,
					Error: &ApiError{
						Code:    "EXTERNAL_SERVICE_NOTIFICATION_FAILED",
						Message: "Failed to notify external service",
					},
				},
			}, nil
		}
		if err := t.api.Transition(waitForEventFSMRetry); err != nil {
			return nil, err
		}
		return &ExecutionResponse{
			Message: "Notified external service, waiting for callback",
			ApiResponse: &ApiResponse{
				Success: true,
			},
		}, nil
	case waitForEventFSMComplete:
		// Persist the raw response to Local Store for rendering
		if err := t.api.WriteToLocalStore("eventResponse", request.Content); err != nil {
			slog.Warn("failed to write callback response to local store", "taskId", t.api.GetTaskID(), "error", err)
		}

		var globalContextPairs map[string]any
		if t.config.Submission != nil && t.config.Submission.Response != nil && t.config.Submission.Response.Mapping != nil {
			parsed, err := t.parseResponseData(request.Content, t.config.Submission.Response.Mapping)
			if err != nil {
				slog.Warn("failed to parse some callback response data fields, continuing with what was found",
					"taskId", t.api.GetTaskID(), "error", err)
			}
			globalContextPairs = parsed
		}

		if err := t.api.Transition(waitForEventFSMComplete); err != nil {
			return nil, err
		}
		return &ExecutionResponse{
			Outputs: globalContextPairs,
			Message: "Task completed by external service",
			ApiResponse: &ApiResponse{
				Success: true,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported action %q for WaitForEventTask", request.Action)
	}
}

// notifyExternalService sends task information to the configured external service with retry logic
func (t *WaitForEventTask) notifyExternalService(ctx context.Context, taskID string, workflowID string) error {
	if t.config.Submission == nil {
		return fmt.Errorf("submission config is missing")
	}

	target := t.config.Submission.Url
	serviceID := t.config.Submission.ServiceID

	extReq := WaitForEventExternalServiceRequest{
		WorkflowID: workflowID,
		TaskID:     taskID,
		ServiceURL: strings.TrimRight(t.serviceBaseURL, "/") + TasksAPIPath,
		Data:       t.resolveInputData(ctx),
	}

	if t.config.Submission.Request != nil {
		extReq.TaskCode = t.config.Submission.Request.TaskCode
	}
	if extReq.TaskCode == "" {
		return fmt.Errorf("taskCode is required for external service submission. Please configure submission.request.taskCode in the plugin config")
	}

	req := remote.Request{
		Method: "POST",
		Path:   target,
		Body:   extReq,
		Retry:  &remote.DefaultRetryConfig,
	}

	// 1. Try to use the Manager if it's available.
	if t.remoteManager == nil {
		return fmt.Errorf("remote manager not initialized")
	}
	// Manager.Call will attempt to resolve the service ID from the Path if serviceID is empty.
	if err := t.remoteManager.Call(ctx, serviceID, req, nil); err != nil {
		return fmt.Errorf("failed to notify external service %q: %w", target, err)
	}
	return nil
}

// getDisplay returns the appropriate title and description based on the current state of the task and the display configuration.
func (t *WaitForEventTask) getDisplay(state waitForEventState) (*WaitForEventDisplay, error) {
	if t.config.Display == nil {
		return nil, fmt.Errorf("display configuration is missing")
	}

	var resolvedState DisplayState
	switch state {
	case notifiedService:
		resolvedState = DisplayStateWaiting
	case notifyFailed:
		resolvedState = DisplayStateFailed
	case receivedCallback:
		resolvedState = DisplayStateCompleted
	default:
		return nil, fmt.Errorf("unsupported wait_for_event plugin state %q", state)
	}

	resolvedDisplay := &WaitForEventDisplay{}
	title, err := resolveDisplayField(t.config.Display.Title, resolvedState, "title")
	if err != nil {
		return nil, err
	}
	resolvedDisplay.Title = title

	description, err := resolveDisplayField(t.config.Display.Description, resolvedState, "description")
	if err != nil {
		return nil, err
	}
	resolvedDisplay.Description = description

	return resolvedDisplay, nil
}

func resolveDisplayField(field any, state DisplayState, fieldName string) (any, error) {
	switch values := field.(type) {
	case nil:
		return nil, nil
	case string:
		return values, nil
	case map[string]any:
		value, exists := values[string(state)]
		if !exists {
			return nil, fmt.Errorf("%s for state %q not found in display configuration", fieldName, state)
		}
		strValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for display %s at state %q, expected string", fieldName, state)
		}
		return strValue, nil
	default:
		return nil, fmt.Errorf("invalid type for display %s, expected string or map[string]any", fieldName)
	}
}

// resolveInputData builds a data map by looking up values from global store based on Template
func (t *WaitForEventTask) resolveInputData(ctx context.Context) any {
	if t.config.Submission == nil || t.config.Submission.Request == nil || len(t.config.Submission.Request.Template) == 0 {
		return nil
	}

	var template any
	if err := json.Unmarshal(t.config.Submission.Request.Template, &template); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal submission template", "error", err)
		return nil
	}

	return jsonutils.ResolveTemplate(template, func(path string) any {
		return t.lookupValueFromGlobalStore(ctx, path)
	})
}

// lookupValueFromGlobalStore retrieves a value from global store using a direct key
func (t *WaitForEventTask) lookupValueFromGlobalStore(_ context.Context, key string) interface{} {
	if key == "" {
		return nil
	}

	value, found := t.api.ReadFromGlobalStore(key)
	if !found {
		return nil
	}

	return value
}

// parseResponseData extracts fields from response data based on mapping.
// It supports nested paths in the response using dot notation.
func (t *WaitForEventTask) parseResponseData(content any, mapping map[string]string) (map[string]any, error) {
	if content == nil || len(mapping) == 0 {
		return nil, nil
	}

	// Ensure content is a map[string]any
	var responseData map[string]any
	switch v := content.(type) {
	case map[string]any:
		responseData = v
	default:
		b, err := json.Marshal(content)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &responseData); err != nil {
			return nil, err
		}
	}

	parsedData := make(map[string]any)
	var missingFields []string

	for responsePath, targetKey := range mapping {
		// Use jsonform utility to handle potential nested paths in the response JSON
		val, exists := jsonform.GetValueByPath(responseData, responsePath)
		if !exists {
			missingFields = append(missingFields, responsePath)
			continue
		}
		parsedData[targetKey] = val
	}

	if len(missingFields) > 0 {
		return parsedData, fmt.Errorf("expected response field(s) not found: %s", strings.Join(missingFields, ", "))
	}

	return parsedData, nil
}

// attachFormDisplay fetches a form definition and attaches it to content under contentKey,
// using data read from storeKey as the formData. A no-op if formID or stored data is absent.
func (t *WaitForEventTask) attachFormDisplay(ctx context.Context, content map[string]any, storeKey, formID, contentKey string) {
	if formID == "" {
		return
	}
	data, err := t.api.ReadFromLocalStore(storeKey)
	if err != nil || data == nil {
		return
	}
	def, err := t.formService.GetFormByID(ctx, formID)
	if err != nil {
		slog.Warn("failed to fetch display form definition", "displayFormId", formID, "error", err)
		return
	}
	content[contentKey] = map[string]any{
		"title":    def.Name,
		"uiSchema": def.UISchema,
		"schema":   def.Schema,
		"formData": data,
	}
}
