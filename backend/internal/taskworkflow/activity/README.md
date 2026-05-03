# Task Workflow Activities

The `activity` package contains activity executors used by the task-as-workflow runtime under `backend/internal/taskworkflow`.

Each activity executor receives:
- `Config`: activity-specific configuration loaded from the workflow node template
- `Inputs`: runtime values mapped into the node by the workflow engine

Each executor returns a structured `Result`. On successful execution, `Result.Outputs` is written back into workflow state through the node's `output_mapping`.

## Execution Contract

All activities implement:

```go
type Executor interface {
    Name() ActivityType
    Execute(ctx context.Context, request Request) (*Result, error)
}
```

`Request` contains:

```go
type Request struct {
    WorkflowID string
    RunID      string
    NodeID     string
    Config     map[string]any
    Inputs     map[string]any
}
```

`Result` contains:

```go
type Status string

const (
    StatusSucceeded Status = "SUCCEEDED"
    StatusFailed    Status = "FAILED"
    StatusRetryable Status = "RETRYABLE"
)

type Result struct {
    Outputs       map[string]any
    RenderPayload map[string]any
    Status        Status
    PersistRender bool
    Message       string
}
```

## Result Semantics

- Return `error` for technical failures such as invalid config, template resolution problems, request construction failures, marshalling failures, or transport failures.
- Return `Result{Status: StatusSucceeded, ...}` for successful execution.
- `Result.Outputs` is the only field currently consumed by the activity handler when completing a workflow node.
- `RenderPayload`, `PersistRender`, and `Message` are part of the activity contract now but are not yet persisted or forwarded by the handler.
- Non-success statuses are currently treated as handler errors until a dedicated failure/update path is added.

## Templating Rules

Activities that build structured payloads use `backend/pkg/jsonutils/resolver.go`.

Supported placeholder forms:
- `"$name"`: whole-value replacement
- `"${student.grade}"`: whole-value replacement with dotted path key
- `"prefix-${version}"`: string interpolation

Lookup is currently performed against `request.Inputs` by exact key match.

Example input map:

```json
{
  "workflow_id": "wf-123",
  "task_id": "task-456",
  "page": 2
}
```

Example template:

```json
{
  "id": "$task_id",
  "label": "task-${page}"
}
```

Resolved output:

```json
{
  "id": "task-456",
  "label": "task-2"
}
```

## JSON_BUILDER

`JSON_BUILDER` resolves a configured template against `Inputs` and marshals the resolved structure to JSON.

### Config

```json
{
  "template": {
    "request": {
      "id": "$workflow_id",
      "name": "$name",
      "label": "workflow-${version}"
    }
  },
  "outputKey": "request_body"
}
```

Fields:
- `template`: any JSON-like structure to resolve and marshal
- `outputKey`: optional result key; defaults to `"json"`

### Behavior

- `template` is required
- the template is resolved with placeholders using `Inputs`
- the resolved structure is marshalled to JSON
- the output value is returned as `json.RawMessage`
- successful execution returns `StatusSucceeded`

### Result

Example result:

```json
{
  "outputs": {
    "request_body": "{\"request\":{\"id\":\"wf-123\",\"name\":\"export\",\"label\":\"workflow-2\"}}"
  },
  "status": "SUCCEEDED"
}
```

In Go, `outputs.request_body` is returned as `json.RawMessage`.

## REST_CALLER

`REST_CALLER` builds an outbound HTTP request from templated config, performs the call through `backend/pkg/remote`, and returns the decoded response.

### Config

```json
{
  "serviceId": "customs-asycuda",
  "method": "POST",
  "path": "/api/consignments/${consignment_id}/documents/${document_type}",
  "query": {
    "version": "$api_version",
    "includeMeta": "$include_meta"
  },
  "headers": {
    "X-Correlation-Id": "$workflow_id"
  },
  "body": {
    "request": {
      "assessmentNo": "$assessment_no"
    }
  },
  "outputKey": "customs_response"
}
```

Fields:
- `serviceId`: optional remote registry service ID
- `method`: optional HTTP method; defaults to `GET`
- `path`: required request path, supports placeholders
- `query`: optional query parameter template
- `headers`: optional header template
- `body`: optional JSON body template
- `outputKey`: optional result key; defaults to `"api_response"`

### Behavior

- `path`, `query`, `headers`, and `body` are resolved against `Inputs`
- query values are stringified into URL query parameters
- header values are stringified into HTTP headers
- body remains structured and is passed to the remote client
- response payload is decoded into `any` and returned under `outputKey`
- successful execution returns `StatusSucceeded`

### Current Limitation

`REST_CALLER` currently uses the JSON request path in `backend/pkg/remote`.

That means:
- request bodies are JSON-marshalled
- `Content-Type: application/json` is used
- raw XML, SOAP envelopes, and other non-JSON wire formats are not supported yet

## Adding New Activities

When adding a new activity:
- keep parsing and execution logic inside this package
- define a small config struct per activity
- add unit tests for config parsing and execution behavior
- return a `Result` with an explicit `Status`
- document the config contract in this README
