# Form Service

The Form Service is a **pure domain service** that provides a simple interface for retrieving form definitions by UUID. It has no knowledge of tasks, task types, or task configurations. **FormService does not expose any HTTP endpoints** - all form access is handled through TaskManager.

## Architecture

```
FormService (Pure Domain Service - No HTTP Endpoints)
  ↓
GetFormByID(formID uuid.UUID) → Returns JSON Forms Schema

TaskManager (Orchestrator)
  ↓
POST /api/tasks/{taskId} → Gets Task → Extracts formID (UUID) from Task.Config → Calls FormService.GetFormByID(formID)
```

**Key Principles:**
- FormService only works with form UUIDs, has no knowledge of tasks
- FormService does not expose any API endpoints
- TaskManager orchestrates: all form access goes through TaskManager via `POST /api/tasks/{taskId}`
- Separation of Concerns: FormService handles forms, TaskManager handles tasks and HTTP

## Usage

### Backend (FormService)

```go
// Initialize service
formService := form.NewFormService(db)

// Get form by UUID (used internally by TaskManager)
formID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
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

## Example Forms

Three example forms are provided:

1. **customs-declaration** - For declaring customs information
   - Fields: consignmentType, traderId, items (array), declarationDate, portOfEntry
   - Tailored to `Consignment` and `Item` models

2. **tea-permit** - For tea export permit applications
   - Fields: applicantName, teaType, quantity, destinationCountry, etc.
   - Example of a trader form

3. **oga-review** - For OGA officer reviews
   - Fields: reviewerName, decision, comments, rejectionReason
   - Example of an OGA form with conditional fields

## Form Structure

Forms follow the JSON Forms format:
- **Schema**: JSON Schema for data validation (https://json-schema.org/)
- **UISchema**: UI Schema for layout and rendering hints (https://jsonforms.io/docs/uischema)

## Seeding Forms

To load example forms into the database:

```go
import "github.com/OpenNSW/nsw/internal/form"

err := form.SeedForms(db, "./backend/internal/form/examples")
if err != nil {
    log.Fatal(err)
}
```

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
