# Form Service

The Form Service is a **pure domain service** that provides a simple interface for retrieving form definitions by UUID. It reads forms from the filesystem (JSON files). It has no knowledge of tasks, task types, or task configurations. **FormService does not expose any HTTP endpoints** - all form access is handled through TaskManager.

## Architecture

```
FormService (Pure Domain Service - No HTTP Endpoints)
  ↓
GetFormByID(formID string) → Reads {formID}.json from configs/forms → Returns JSON Forms Schema

TaskManager (Orchestrator)
  ↓
POST /api/tasks/{taskId} → Gets Task → Extracts formID (UUID) from Task.Config → Calls FormService.GetFormByID(formID)
```

**Key Principles:**
- FormService reads form definitions from JSON files on the filesystem
- FormService only works with form UUIDs, has no knowledge of tasks
- FormService does not expose any API endpoints
- TaskManager orchestrates: all form access goes through TaskManager via `POST /api/tasks/{taskId}`
- Separation of Concerns: FormService handles forms, TaskManager handles tasks and HTTP

## Usage

### Backend (FormService)

```go
// Initialize service with path to forms directory
formService := form.NewFormService("configs/forms")

// Get form by UUID (used internally by TaskManager)
formID := "550e8400-e29b-41d4-a716-446655440000"
formResponse, err := formService.GetFormByID(ctx, formID)
if err != nil {
    // Handle error
}

// formResponse contains:
// - ID: UUID of the form
// - Schema: JSON Schema for validation
// - UISchema: UI Schema for layout
// - Name, Version
```

### Backend (TaskManager - for task-related operations)

```go
// TaskManager uses FormService internally
// When a portal calls POST /api/tasks/{taskId}:
// 1. TaskManager gets the task
// 2. Extracts formID (UUID) from Task.Config
// 3. Calls FormService.GetFormByID(formID)
// 4. Returns form to portal
```

### Frontend (Portal)

```typescript
// Portal receives Task object from Workflow Manager
// Task has: { id: taskId, type: "TRADER_FORM", config: { formId: "uuid-here" }, ... }

// Fetch form using taskID (handled by TaskManager)
const response = await fetch(`/api/tasks/${task.id}`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({})
});
const formData = await response.json();

// Use with JSON Forms
import { JsonForms } from '@jsonforms/react';

<JsonForms
  schema={formData.schema}
  uischema={formData.uiSchema}
  data={{}} // Start with empty data or provide an initial data object
  onChange={({ data }) => {
    // Handle form data changes
  }}
/>
```

## Form Structure

Forms are stored as JSON files in the configured directory (default `configs/forms`). Each file should be named `{formID}.json`.

Example file `configs/forms/550e8400-e29b-41d4-a716-446655440000.json`:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Customs Declaration Form",
  "schema": { ... },
  "uiSchema": { ... },
  "version": "1.0"
}
```

## Configuration

The forms directory can be configured via environment variable:
`FORMS_CONFIG_PATH=configs/forms`

## API Endpoints

**Note:** FormService does not expose any HTTP endpoints. All form access is handled through TaskManager.

### POST /api/tasks/{taskId} (TaskManager Handler)

Returns the form definition for a task. Extracts formID (UUID) from Task.Config automatically. This endpoint is handled by TaskManager, which orchestrates the call to FormService.

**Request:**
- `taskId`: UUID of the task (portals already have this)
- Method: POST

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Customs Declaration Form",
  "schema": { ... },
  "uiSchema": { ... },
  "version": "1.0"
}
```

**Use Case:** Portals working on a task - they only need the taskID.

**Handler:** `TaskManager` → Gets Task → Extracts formID (UUID) from Task.Config → `FormService.GetFormByID(formID)`

## References

- [JSON Forms Documentation](https://jsonforms.io/)
- [JSON Forms Examples](https://jsonforms.io/examples/basic)
- [JSON Schema Specification](https://json-schema.org/)
