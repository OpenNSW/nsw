# `render.json` — Authoring Guide & Spec

This document is the complete reference for the `render.json` config that drives
task rendering in the trader-app. It exists so that, given only this file, a
human or an AI agent can author a new task's `render.json` (and, in the
extender section, add a new renderer type) without reading the source.

It has two audiences:

- **Config author** (most readers). Writes `render.json` for a new task type or
  edits an existing one. Touches no Go, no React. Sections 1–11 cover this.
- **Framework extender** (rare). Adds a new `ZoneComponent` type — e.g., a
  `TABLE` renderer, a `SIGNATURE_PAD`. Touches both backend (new projector)
  and frontend (new React renderer). Section 12 is the end-to-end checklist;
  it assumes the config-author sections have been read.

The architecture this document describes shipped in PR #573 (`feat/395`). If
the code diverges from the spec, the code wins — update this doc.

---

## 1. Conceptual model

A `render.json` describes one task type's UI. It is loaded once at process
start by `backend/internal/taskv2/registry.LoadConfigsInto` and held in memory;
every `GET /api/v1/tasks/{id}` response for that task type is reassembled from
it against the task's current `state` and `data`.

Four layers participate. Each owns one fact; nothing else owns that fact.

| Layer                          | Owns                                                                                                  | Lives in                                                                      |
|--------------------------------|-------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------|
| **Projector blueprint**        | Which zone exists, what projector produces its payload, what data it sees, when it is visible.        | `sections.<zone>` (top-level keys; see §3).                                   |
| **Section (handle claim)**     | Which commands this zone claims responsibility for, and which renderer element each command binds to. | `sections.<zone>.handles[]`.                                                  |
| **State (action legality)**    | Which commands are legal *right now*, given the task's state.                                         | `states.<STATE>.actions[]`.                                                   |
| **Renderer (element catalog)** | Physical presentation of each handle (button style, position, gating).                                | Frontend renderer source — e.g., `FormRenderer.tsx`'s `FORM_ELEMENT_CATALOG`. |

The merge rule that ties them together (see §2.3): a handle survives to the
wire iff its `command` appears in the current state's `actions[]`. Everything
else — interactivity, read-only mode, footer visibility — is **derived** from
"does this zone have at least one surviving handle?". There is no `role`,
no `interactive` flag, no `isWorkspace` boolean. The data answers the question.

### 1.1 Why this split

The same form may need to be editable in state `PENDING_USER`, read-only in
state `UNDER_REVIEW`, and absent in state `COMPLETED`. The split lets each
concern move independently:

- Add a new state? Edit `states.<NEW_STATE>`. The form's zone definition is
  untouched.
- Add a new button to the form? Edit `sections.workspace.handles[]` and the
  state(s) that should make it legal. The renderer is untouched.
- Restyle the submit button? Edit the renderer's element catalog. No config
  change.

---

## 2. The wire

This is what the trader-app receives on `GET /api/v1/tasks/{id}`. Everything
in §3–§6 is in service of producing this. If a config change doesn't change
this output for any (state, data) combination, it doesn't change behaviour.

### 2.1 Shape

```jsonc
{
  "task_id":    "string",
  "task_type":  "string",
  "state":      "STATE_NAME",
  "created_at": "RFC3339 timestamp",
  "updated_at": "RFC3339 timestamp",

  // Optional UI banner; not set from render.json — plugin/orchestrator owned.
  "alert":  "string | { message, title?, variant? }",
  // Optional activity log; not set from render.json.
  "audit":  [ { timestamp, actor, event, from_state?, to_state?, details? } ],

  // The per-zone projector output, merged with state-filtered handles.
  // Keys are the same zone slot keys used in render.json's `sections`.
  "view": {
    "<zone-slot>": {
      "type":     "FORM | MARKDOWN | REDIRECT | RAW | <custom>",
      "handles":  [ { "command", "label", "element"? } ],   // optional; omitted when empty
      "payload":  { /* shape depends on type — see §6 */ }
    }
  }
}
```

There is **no top-level `actions` field**. There is **no `role` per zone**.
Operations ship inside `view.<zone>.handles[]`, gated to the current state.

### 2.2 A real response

For `state: "PENDING_USER"` on the `2-payment_app_fee` task (config in §3),
the `view` field contains exactly one zone:

```json
{
  "view": {
    "workspace": {
      "type": "FORM",
      "handles": [
        { "command": "submit", "label": "Proceed to Payment", "element": "primary_action" }
      ],
      "payload": {
        "schema":   { /* JSONForms schema from the template */ },
        "uiSchema": { /* optional UI hints */ },
        "data":     { /* current values plucked via dataKey */ }
      }
    }
  }
}
```

For the same task in `state: "PENDING_PAYMENT"`, the `view` field contains
two zones (`payment_instructions` and `payment_details`), neither with
handles, because that state declares `actions: []`. The form renders
read-only; no buttons show.

### 2.3 How the wire is built

The HTTP handler calls `ZoneViewAssembler.Assemble`
(`backend/internal/taskv2/renderer/zone_assembler.go`). It runs two passes
over the same `render.json` bytes:

1. **Projection pass** (uiprojector). Decodes the JSON as a `Blueprint`,
   iterates `sections`, applies `visibleWhen`, fetches each section's
   `templateId`, plucks data via `dataKey`, hands both to the named projector,
   and emits a `map[slot] → {type, payload}`.

2. **Merge pass** (zone_assembler). Decodes the same JSON as a
   `TaskTemplateConfig` (a thinner view that ignores projector fields),
   reads `states[<currentState>].actions[]` to build the legal-command set,
   then for each slot from pass 1 attaches the subset of
   `sections.<slot>.handles[]` whose `command` is in that set.

The two decodes are deliberate: each parser ignores fields it doesn't own.
A field new to one parser causes no churn in the other.

A slot present in the view but missing from `sections` gets no handles
(renders with `handles` omitted). A handle whose `command` isn't in the
current state's `actions[]` is dropped silently. If the merged handle list
is empty, `handles` is omitted from the wire entirely (not sent as `[]`).

---

## 3. Canonical annotated example

Real file: `backend/configs/fcau/2-payment_app_fee/render.json`. This task is
the most representative — three projector types, multi-state visibility,
handles with state gating.

```jsonc
{
  // §3.1 — top-level identity
  "id":   "fcau-pay-app-fee-flow:render",
  "type": "PAYMENT",

  // §3.2 — sections: the zones this task can render
  "sections": {

    // Zone slot "workspace". Slot keys are arbitrary strings but three are
    // privileged for display ordering in TraderZoneLayout: "instructions",
    // "workspace", "reference" — they render first, in that order. All
    // other slots render after, in JSON insertion order.
    "workspace": {
      "templateId": "fcau-pay-app-fee--select-method-form",  // §4.1
      "title":      "Select Payment Gateway",                // §4.2
      "projector":  "FORM",                                  // §4.3
      "dataKey":    "payment_method",                        // §4.4
      "visibleWhen": {                                       // §4.5
        "states": ["PENDING_USER"]
      },
      "handles": [                                           // §4.6
        { "command": "submit", "label": "Proceed to Payment", "element": "primary_action" }
      ]
    },

    // Zone slot "payment_instructions". Visible only in PENDING_PAYMENT;
    // uses the custom PAYMENT projector, which switches between
    // MARKDOWN and REDIRECT output based on the selected payment method.
    "payment_instructions": {
      "templateId":  "fcau-pay-app-fee--instructions-wrapper",
      "title":       "Payment Instructions",
      "projector":   "PAYMENT",
      "dataKey":     "payment",
      "visibleWhen": { "states": ["PENDING_PAYMENT"] }
      // No handles[] — this zone is informational in every state where it
      // is visible.
    },

    "payment_details": {
      "templateId":  "fcau-pay-app-fee--payment-form",
      "title":       "Application Fee Details",
      "projector":   "FORM",
      "dataKey":     "payment",
      "visibleWhen": { "states": ["PENDING_PAYMENT"] }
      // No handles[]. The form will render read-only — see §6.1.
    },

    "payment_success": {
      "templateId":  "fcau-pay-app-fee--success-message",
      "title":       "Payment Successful",
      "projector":   "MARKDOWN",
      "dataKey":     "payment",
      "visibleWhen": { "states": ["COMPLETED"] }
    }
  },

  // §3.3 — states: which commands are legal in each state.
  // A state listed here with an empty actions[] means "visible but no
  // operations". A state omitted entirely is treated the same as
  // actions: [] — every handle drops, every zone renders passive.
  "states": {
    "PENDING_USER": {
      "actions": [
        { "command": "submit" }
      ]
    },
    "PENDING_PAYMENT": {
      "actions": []
    }
  }
}
```

### 3.1 Top-level fields

| Field      | Type   | Required | Purpose                                                                                                                                                |
| ---------- | ------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `id`       | string | **yes**  | Globally unique identifier for this render config. Registered in the in-memory template registry; referenced by `TaskTemplate.RenderConfigID`.         |
| `type`     | string | **yes**  | Free-form label for the task type (e.g. `APPLICATION`, `REVIEW`, `PAYMENT`, `CERTIFICATE_ISSUANCE`). Used as `TaskTemplate.Type`; not interpreted by the renderer. |
| `sections` | object | **yes**  | Map of zone slot key → section blueprint. See §4. At least one entry expected.                                                                         |
| `states`   | object | no       | Map of state name → state declaration. See §5. Omit entirely if the task has no interactive states (every zone renders passive regardless of state).   |

### 3.2 Slot key conventions

Slot keys are arbitrary, but `TraderZoneLayout` (`portals/apps/trader-app/src/zones/TraderZoneLayout.tsx`) gives three keys privileged display order:

```ts
const ZONE_ORDER = ['instructions', 'workspace', 'reference']
```

These render first, in that order, when present. Every other slot renders
after, in JSON insertion order. There is **no semantic difference** between
"instructions" and "anything else" — the keys are just display hooks. A zone
named "workspace" is not automatically interactive; interactivity is derived
from handle legality, not slot key.

### 3.3 State name conventions

State names are uppercase snake-case by convention (`PENDING_USER`,
`PENDING_PAYMENT`, `QUEUED_EXTERNALLY`, `COMPLETED`). They are compared
**case-insensitively** against `facts.State` (the live task state). They have
no enum — whatever the workflow engine emits as the state is what `render.json`
must match.

---

## 4. Section fields (the projector blueprint)

Every entry under `sections` is a `SectionBlueprint`. The Go type is in
`backend/pkg/uiprojector/blueprint.go`; this is the field-by-field spec.

### 4.1 `templateId` — required, string

Identifies the template content to render. Resolved at projection time via
the in-memory template registry; the value here must match an `id` field of
a registered template file (typically `*_jsonform.json` or another JSON
template alongside `render.json`).

Missing or unresolved → `assembler: failed to fetch template <id>` (HTTP 500).

### 4.2 `title` — optional, string

Display title for the section. **Currently unused** by the trader-app —
`Zone.tsx` renders the slot key in the header, not the title. The field is
preserved in the projection pipeline (`Section.Title`) for future use; safe
to omit. Existing configs include it; new configs may.

### 4.3 `projector` — required, string

Names the projector that produces this zone's payload. Built-in values:

| Projector  | Wire `type`              | Payload shape                                                                                                                                            |
| ---------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FORM`     | `FORM`                   | `{ schema, uiSchema?, data? }` — see §6.1.                                                                                                               |
| `MARKDOWN` | `MARKDOWN`               | `{ content: "<rendered markdown>" }` — see §6.2.                                                                                                         |
| `RAW`      | `RAW`                    | The data plucked via `dataKey`, unchanged.                                                                                                               |
| `PAYMENT`  | `MARKDOWN` *or* `REDIRECT` | Switches at projection time based on the selected payment method's `type`; emits `{ content }` for description methods, `{ checkout_url, content }` for redirect. See §6.3. |

The wire `type` is what the frontend dispatches on, not the projector name.
A custom projector may emit any wire type (multiple, even); see §12.

Unknown projector → `assembler: unknown projector <name>` (HTTP 500).

### 4.4 `dataKey` — optional, string

Names the key in `facts.Data` whose value is passed to the projector. The
plucked value's shape is projector-specific:

- `FORM` — any JSON value. Used as the form's `data` (initial values).
- `MARKDOWN` — a map passed as `text/template` context.
- `PAYMENT` — must be a map; see §6.3 for required keys.

If `dataKey` is omitted, the projector receives the entire `facts.Data`
map. (`FormProjector` will accept this without error and use it as `data`;
some custom projectors may not.)

If `dataKey` names a key that's not present in `facts.Data`, the projector
receives `nil`. For `FORM` this means the form renders with no initial
values — usually fine. For `MARKDOWN` it means the template renders with
`nil` context — `{{ .Field }}` will print `<no value>`.

### 4.5 `visibleWhen` — optional, object

Declarative visibility. Both rules below are AND-ed; if either fails, the
section is excluded from the projection output entirely (not just hidden in
CSS — it never enters the `view` map). Omit the whole object to mean "always
visible".

```jsonc
"visibleWhen": {
  "states":         ["STATE_A", "STATE_B"],  // optional
  "requireDataKey": "some_key"               // optional
}
```

- `states` — case-insensitive list of states in which this section renders.
  If present, the current `facts.State` must match one of them.
- `requireDataKey` — name of a key that must exist in `facts.Data` *and*
  whose value must not be `nil`. Use this for sections that should appear
  only after a particular piece of data has been written by the workflow
  (e.g. a reviewer form that only appears once reviewer input has been
  collected at least once).

Implementation: `ShouldRender` in `backend/pkg/uiprojector/visibility.go`.

### 4.6 `handles` — optional, array

Each handle declares: "this section is responsible for the `<command>`
operation; when its element fires, dispatch that command." Handles are
**claims**, not buttons — whether they render as buttons (and how) is
the renderer's decision based on `element`.

```jsonc
"handles": [
  { "command": "submit",       "label": "Submit application", "element": "primary_action" },
  { "command": "save_draft",   "label": "Save draft",         "element": "secondary_action" },
  { "command": "reject",       "label": "Reject",             "element": "danger_action" }
]
```

| Field     | Type   | Required | Purpose                                                                                                                                                              |
| --------- | ------ | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `command` | string | **yes**  | Identifies the operation. Joined to `states.<STATE>.actions[].command` to determine legality. Sent on the wire and to `onAction` in the frontend.                    |
| `label`   | string | **yes**  | User-facing button text.                                                                                                                                             |
| `element` | string | no       | Renderer-owned identifier for *how* this handle is presented. For FORM zones: `primary_action`, `secondary_action`, `danger_action` (see §6.1). Unknown → solid grey button. |

**One section can claim N handles.** A FORM with both `submit` and `save_draft`
is fine — both render in the form's footer when both are legal.

**N sections can claim the same command.** Two zones each declaring
`{command: "submit"}` will each render their own button; both fire the same
backend command. Useful when the same operation is reachable from multiple
visual contexts; uncommon.

**Handles never render outside their declaring section.** There is no global
toolbar fed by handles. A button must be claimed by *some* zone.

If `handles` is omitted or empty, the section renders passive — no footer,
no buttons. For a FORM zone, this also makes the form read-only (§6.1).

---

## 5. State fields (action legality)

```jsonc
"states": {
  "PENDING_USER": {
    "actions": [
      { "command": "submit" },
      { "command": "save_draft" }
    ]
  },
  "UNDER_REVIEW": { "actions": [] }
}
```

Each entry under `states` is a `StateView`. Field-by-field:

| Field     | Type  | Required | Purpose                                                                                                                                                         |
|-----------|-------|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `actions` | array | no       | List of commands that are legal in this state. Each entry is `{ "command": "<id>" }` — no other fields. Empty array or omitted state → no operations are legal. |

The shape is intentionally narrow: state owns legality and **only** legality.
No `label`, no `variant`, no `element` — those are renderer-side. No `kind` —
the command name is the identifier.

### 5.1 Missing state

If `facts.State` doesn't match any key in `states`, the assembler treats it
as having no actions: every handle drops, every zone renders passive. This
is the right behaviour for terminal states (`COMPLETED`, `REJECTED`) — no
need to declare them as empty entries, though doing so is fine and documents
intent.

### 5.2 Action present, no claiming section

If a state declares `{command: "approve"}` but no section's `handles[]`
contains `approve`, the action is unreachable (no button anywhere). No error
is raised — the assembler doesn't check. This is by design: the workflow
engine may receive `approve` via other channels (an external API, a webhook),
and the UI just doesn't surface it.

### 5.3 Handle present, no legal action

The reverse — a section claims `cancel` but no state's `actions[]` includes
it — means `cancel` is *never* exposed in the UI. The handle drops in every
state. Useful as documentation that "this section *could* handle cancel if
you ever make it legal", but otherwise dead config; consider deleting.

---

## 6. Shipped renderers — payload catalog

For each wire `type` the trader-app understands, this section documents the
payload shape it expects, the elements its renderer recognises, and the
behaviour the renderer derives.

### 6.1 `FORM` (`portals/apps/trader-app/src/zones/renderers/FormRenderer.tsx`)

**Payload** (produced by `uiprojector.FormProjector`):

```ts
{
  schema:   JsonSchema      // JSONForms schema for fields + validation
  uiSchema?: UISchemaElement // optional JSONForms UI hints
  data?:    Record<string, unknown>  // current values plucked via dataKey
}
```

**Source of `schema` / `uiSchema`.** Both come from the template file named
in `templateId`. The template is a JSON document with the shape
`{ id, title, schema, uiSchema? }`; `FormProjector` extracts `schema` and
`uiSchema` and hands them through. Convention: filenames end in
`_jsonform.json` so the loader recognises them.

**Source of `data`.** The value plucked from `facts.Data` via `dataKey`. If
`dataKey` is omitted, the whole `Data` map is used.

**Element catalog.** Defined inline in `FormRenderer.tsx`:

```ts
const FORM_ELEMENT_CATALOG = {
  primary_action:   { variant: 'solid' },
  secondary_action: { variant: 'outline' },
  danger_action:    { variant: 'solid', color: 'red' },
}
```

A handle whose `element` is one of these gets that visual treatment. An
unknown `element` (or no `element` at all) falls back to plain solid.
Buttons render in the order they appear in `handles[]`, right-aligned in
the form's sticky footer.

**Derived behaviour.**

- The form is **editable** iff it has at least one handle on the wire (i.e.
  the section claims a handle whose command is legal in the current state)
  **and** the renderer was passed an `onAction` callback. Otherwise it
  renders **read-only** (`JsonForms readonly={true}`), with no footer.
- Submit is **disabled** when the form's required fields aren't all filled
  or when the schema reports validation errors. The button text becomes
  `Submitting…` while a dispatch is in flight.
- Validation is structural only (JSONForms + required-field walk); business
  rules belong on the backend.

### 6.2 `MARKDOWN` (`portals/apps/trader-app/src/zones/renderers/MarkdownRenderer.tsx`)

**Payload:**

```ts
{ content: string }
```

The string is rendered by `react-markdown` with a curated component map
(headings, lists, links, code, blockquotes). No element catalog; the renderer
ignores `handles` even if present. If you want a Markdown zone with buttons,
that is a renderer-side change — extend `MarkdownRenderer` to accept and
render handles, or fork it into a new renderer (§12).

**Source of `content`.** `uiprojector.MarkdownProjector` reads the template
file, treating it either as a raw markdown string or as
`{ "template": "<markdown body with {{ .Fields }}>" }` (auto-detected), and
runs it through Go `text/template` with the data plucked via `dataKey`.

### 6.3 `REDIRECT` (`portals/apps/trader-app/src/zones/renderers/RedirectRenderer.tsx`)

Produced by the custom `PaymentProjector` (`backend/internal/taskv2/renderer/payment_projector.go`) — not by a built-in projector. Emitted when the selected payment method's `type` is `REDIRECT`.

**Payload:**

```ts
{
  checkout_url: string  // external URL the browser is sent to
  content:      string  // markdown shown on the redirect page itself
}
```

**Derived behaviour.** On first render with a fresh `checkout_url`, the
renderer auto-navigates the window to that URL after setting a
`sessionStorage` key. On subsequent renders (e.g. user navigates back), it
renders a "Return to payment session" button rather than re-redirecting.
A "Reset redirection state" button clears the sessionStorage key for
debugging. The renderer ignores `handles`.

**`PaymentProjector` data contract.** When `projector: "PAYMENT"`, the value
plucked via `dataKey` must be a `map[string]any` with these fields:

| Key                | Used for                                                                                            |
| ------------------ | --------------------------------------------------------------------------------------------------- |
| `selected_method`  | Looked up in the payment methods registry. Defaults to `"lankapay"` if missing/empty.               |
| `reference_number` | Substituted into the method's instructions template as `{{ .ReferenceNumber }}`.                    |
| `amount`           | `{{ .Amount }}`                                                                                     |
| `currency`         | `{{ .Currency }}`                                                                                   |
| `checkout_url`     | Copied verbatim to the wire (for REDIRECT methods) and available as `{{ .CheckoutURL }}` in template. |
| `service_name`     | `{{ .ServiceName }}`                                                                                |
| `service_type`     | `{{ .ServiceType }}`                                                                                |
| `org_name`         | `{{ .OrganizationName }}`                                                                           |

### 6.4 Anything else

A wire `type` that isn't in the renderer registry (`renderers/index.tsx`)
renders via `UnknownRenderer` — a small fallback panel showing the type
string. No error; the zone just doesn't display meaningfully. This is by
design so a backend rolling out a new payload type doesn't break older
trader-app builds; see §12 for the proper extension flow.

---

## 7. Recipes — common shapes

Each recipe is the minimum `render.json` that produces the described
behaviour. Use them as templates; paste, rename, edit.

### 7.1 Single-state form with one submit button

```json
{
  "id":   "myapp-apply-form:render",
  "type": "APPLICATION",
  "sections": {
    "workspace": {
      "templateId": "myapp-apply-form--user-form",
      "title":      "Application",
      "projector":  "FORM",
      "dataKey":    "userform",
      "handles": [
        { "command": "submit", "label": "Submit", "element": "primary_action" }
      ]
    }
  },
  "states": {
    "PENDING_USER": { "actions": [ { "command": "submit" } ] }
  }
}
```

### 7.2 Form that is editable in one state, read-only in another

The same `workspace` section in two states:

```json
{
  "states": {
    "PENDING_USER":  { "actions": [ { "command": "submit" } ] },
    "UNDER_REVIEW":  { "actions": [] }
  }
}
```

The handle is filtered out in `UNDER_REVIEW` (the command isn't legal),
which collapses the form to zero handles, which derives `interactive: false`,
which sets `readonly: true`. No section changes; no renderer changes.

### 7.3 Form with submit + save-draft

```json
"handles": [
  { "command": "submit",     "label": "Submit",     "element": "primary_action"   },
  { "command": "save_draft", "label": "Save draft", "element": "secondary_action" }
]
```

…and in the relevant state:

```json
"PENDING_USER": {
  "actions": [
    { "command": "submit" },
    { "command": "save_draft" }
  ]
}
```

The renderer renders them in handle-array order, right-aligned. `save_draft`
is *not* gated by form validity (the renderer does gate `primary_action` on
the form being valid; `secondary_action` fires regardless) — if you need
different gating, that's a renderer change.

### 7.4 Read-only reviewer panel that appears only after data exists

```json
"reference": {
  "templateId":  "myapp-reviewer-form",
  "title":       "Reviewer Feedback",
  "projector":   "FORM",
  "dataKey":     "reviewerform",
  "visibleWhen": { "requireDataKey": "reviewerform" }
}
```

No `handles[]` → form renders read-only, no footer. Section is omitted from
the projection until `facts.Data["reviewerform"]` is set (and non-nil).

### 7.5 Status banner only in one state

```json
"status_message": {
  "templateId":  "myapp-task--instructions",
  "title":       "Status",
  "projector":   "MARKDOWN",
  "visibleWhen": { "states": ["QUEUED_EXTERNALLY"] }
}
```

`dataKey` omitted → projector receives the whole `Data` map (fine for
templates that don't reference any variables).

### 7.6 Payment redirect

See §3 for the full config. The two-state pattern is:

- State 1 (`PENDING_USER`) — workspace form to pick a method, with a
  `submit` handle.
- State 2 (`PENDING_PAYMENT`) — `payment_instructions` (PAYMENT projector,
  may emit REDIRECT) and `payment_details` (FORM, read-only because no
  handles), both visible only in this state.

---

## 8. Authoring checklist

When you write or edit a `render.json`:

1. **Top-level `id` and `type`.** `id` must be unique across the loaded
   config tree. `type` is free-form; conventionally uppercase snake-case.
2. **Every slot in `sections` has a `templateId` and a `projector`.** Confirm
   `templateId` matches the `id` field of a template file in the same task
   folder (or anywhere that gets loaded into the registry).
3. **Every `handles[].command` appears in some `states.<STATE>.actions[]`,**
   or you've consciously decided to keep it dead. The validator doesn't
   check this; you have to.
4. **Every `states.<STATE>.actions[].command` is claimed by some section's
   `handles[]`,** or it's intentionally backend-only. Same — no validator.
5. **States that exist in the workflow are listed,** unless they should be
   fully passive (terminal). A missing state is treated as `actions: []`.
6. **Section keys match the conventions** if you want them ordered:
   `instructions`, `workspace`, `reference` render first. Otherwise insertion
   order.
7. **`visibleWhen.states` uses the same state names as the workflow.** Names
   are case-insensitive but be consistent.
8. **No `id` on sections** (the map key is identity). No `role`. No
   `kind`/`variant` on handles or actions. Those fields were removed in PR
   #573; old configs may still contain them in git history but they're
   silently ignored.

---

## 9. Folder & loader conventions

Configs live under `backend/configs/<app>/<task-folder>/`. Each immediate
subfolder of `backend/configs/<app>/` is one task. The loader
(`backend/internal/taskv2/registry/config_loader.go`) classifies files by
name:

| Filename pattern                            | Treated as                                 |
| ------------------------------------------- | ------------------------------------------ |
| `workflow.json` or `*_workflow.json`        | Workflow definition (orchestrator).        |
| `render.json`                               | This file. Required, exactly one per task. |
| `*_jsonform.json`                           | JSONForms schema template.                 |
| Anything else `.json`                       | Subtask template.                          |

The loader fails if a task folder lacks either a workflow file or a
`render.json`. The folder name itself is irrelevant — it's the file
classification that matters.

When you add a new task, drop its `render.json` and a `workflow.json` (plus
any `_jsonform.json` templates it references) into a fresh folder under the
relevant app config root. No code change; the loader picks it up at next
process start.

---

## 10. Debugging

Symptom-first. If something on screen doesn't match `render.json`, work
down this table.

| Symptom                                                    | Likely cause                                                                                                                                                             | Check                                                                                                                                               |
|------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| Zone is missing from the page entirely.                    | `visibleWhen` excluded it.                                                                                                                                               | Confirm `facts.State` matches one of `visibleWhen.states` (case-insensitive). Confirm `requireDataKey` is set in `facts.Data` and not nil.          |
| Zone shows up but has no buttons.                          | No handle survived the state filter.                                                                                                                                     | Does `states[<currentState>].actions[]` contain *any* of this section's `handles[].command`? If state isn't in `states` at all, every handle drops. |
| Form is read-only when it should be editable.              | Same as above — zero surviving handles → derived `readonly`.                                                                                                             | Check the wire (`GET /api/v1/tasks/{id}`) for `view.<zone>.handles`; if absent, it's a config issue.                                                |
| Form is editable but submit is greyed out.                 | Required-field gating (frontend) or validation errors from JSONForms.                                                                                                    | Check the form's `schema.required`; check the JSONForms data state.                                                                                 |
| Button has the wrong style.                                | `element` value isn't in the renderer's catalog, or you used an alias the renderer doesn't recognise.                                                                    | For FORM: must be one of `primary_action`, `secondary_action`, `danger_action`, or omit for fallback.                                               |
| Wire says `"type": "FORM"` but no payload shows.           | Template fetch failed silently — but actually it errors out at HTTP 500; if the zone renders empty, more likely the `data` is empty.                                     | Inspect `payload.data` in the wire. Confirm `dataKey` matches a real key in `facts.Data`.                                                           |
| `assembler: unknown projector X`                           | `projector` field names a projector that isn't registered.                                                                                                               | Built-in names are `FORM`, `MARKDOWN`, `RAW`. Custom: `PAYMENT`. Anything else needs §12-extender work.                                             |
| `assembler: failed to fetch template X`                    | `templateId` doesn't match any registered template's `id`.                                                                                                               | Grep for `"id": "<your templateId>"` in the config tree. Check the loader log for which templates registered.                                       |
| `task folder X: render.json missing or has no id`          | Loader couldn't find a `render.json` in the folder, or it had no top-level `id` field.                                                                                   | Add it.                                                                                                                                             |
| Wire shows `view: {}` even though `sections` is populated. | All sections failed `visibleWhen` for the current state.                                                                                                                 | Likely a state-name mismatch (typo, casing — though casing is forgiven).                                                                            |
| Two zones render in unexpected order.                      | Slot keys aren't in `ZONE_ORDER` and you're relying on JSON object key order, which is preserved by Go's `encoding/json` but may be reorganised by editors / formatters. | Use one of `instructions`, `workspace`, `reference` for known positions; accept insertion order for the rest.                                       |

---

## 11. Glossary

- **Action** — a state-side declaration that a command is legal right now.
  Shape: `{ command }`. Carries no presentation.
- **Blueprint** — the projector-pipeline view of `render.json`. The
  uiprojector reads only `sections.<slot>.{templateId, title, projector,
  dataKey, visibleWhen}`.
- **Command** — a string identifier for an operation (`submit`, `approve`,
  `cancel`). Joins handles to actions. Sent to the backend on activation.
- **Element** — a renderer-owned identifier for *how* a handle is presented
  (`primary_action`, `secondary_action`). The data layer doesn't interpret
  it; the renderer's element catalog does.
- **Facts** — the per-render input: `{ State, Data }`. Comes from the task
  record, not from `render.json`.
- **Handle (handle claim)** — a section's binding of a command to a renderer
  element. Shape: `{ command, label, element? }`. Filtered against state
  legality before shipping.
- **Projector** — a backend strategy that turns a template + data into a
  wire payload. Built-in: FORM, MARKDOWN, RAW; custom: PAYMENT.
- **Renderer** — a frontend component that consumes a wire payload of a
  given `type` and turns it into pixels. Owns its element catalog and its
  interactivity rules.
- **Section** — one entry in `render.json`'s `sections` map. Holds both a
  projector blueprint and a list of handle claims.
- **State** — the workflow's current logical status. The key in
  `states.<STATE>`. Compared case-insensitively against `facts.State`.
- **Zone (slot)** — one rendered region of the page. Identified by the
  section key in `render.json`; the same key is used in the wire's `view`
  map and in the frontend's display ordering.
- **Wire** — the JSON document returned by `GET /api/v1/tasks/{id}`. §2.

---

## 12. Extending: adding a new renderer type

This section is for the framework extender. Read §1–§6 first.

A "new renderer" means: a new wire `type` that the frontend understands.
That requires changes in three places (backend projector, wire-type
constant, frontend renderer), all coordinated so the new type is emitted
*and* consumed.

Worked example: adding a `TABLE` renderer that displays tabular data.

### 12.1 Backend — new projector

File: `backend/internal/taskv2/renderer/table_projector.go` (or anywhere
in this package; keeping projectors here groups them).

```go
package renderer

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/OpenNSW/nsw/backend/pkg/uiprojector"
)

// ProjectorTable identifies the table projector.
const ProjectorTable uiprojector.ProjectorType = "TABLE"

type TableProjector struct{}

func NewTableProjector() *TableProjector { return &TableProjector{} }

func (p *TableProjector) Type() uiprojector.ProjectorType { return ProjectorTable }

func (p *TableProjector) Project(
    ctx context.Context, templateContent []byte, data any,
) (uiprojector.Projection, error) {
    // Template content shape: { "columns": [{ "key", "label" }, ...] }
    var tmpl struct {
        Columns []struct {
            Key   string `json:"key"`
            Label string `json:"label"`
        } `json:"columns"`
    }
    if err := json.Unmarshal(templateContent, &tmpl); err != nil {
        return uiprojector.Projection{}, fmt.Errorf("table_projector: parse template: %w", err)
    }
    rows, _ := data.([]any) // data is the array of row objects
    return uiprojector.Projection{
        Type: uiprojector.SectionType("TABLE"),
        Content: map[string]any{
            "columns": tmpl.Columns,
            "rows":    rows,
        },
    }, nil
}
```

### 12.2 Register the projector at wiring time

File: `backend/internal/taskv2/wiring.go`, around line 52:

```go
projectors := append(
    uiprojector.DefaultProjectors(),
    taskrenderer.NewPaymentProjector(paymentService),
    taskrenderer.NewTableProjector(),  // ← add
)
```

### 12.3 Frontend — payload type

File: `portals/apps/trader-app/src/zones/types.ts`:

```ts
export type TablePayload = {
  columns: { key: string; label: string }[]
  rows: Record<string, unknown>[]
}

export type ZoneComponent =
  | (ZoneComponentBase & { type: 'FORM'; payload: FormPayload })
  | (ZoneComponentBase & { type: 'MARKDOWN'; payload: MarkdownPayload })
  | (ZoneComponentBase & { type: 'REDIRECT'; payload: RedirectPayload })
  | (ZoneComponentBase & { type: 'TABLE'; payload: TablePayload })  // ← add
```

### 12.4 Frontend — renderer component

File: `portals/apps/trader-app/src/zones/renderers/TableRenderer.tsx`:

```tsx
import type { ZoneRendererProps } from './types'

export function TableRenderer({ payload }: ZoneRendererProps<'TABLE'>) {
  const { columns, rows } = payload
  return (
    <div className="p-6 overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr>{columns.map((c) => <th key={c.key} className="text-left py-2 px-3">{c.label}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-t border-gray-100">
              {columns.map((c) => <td key={c.key} className="py-2 px-3">{String(r[c.key] ?? '')}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

### 12.5 Register the renderer

File: `portals/apps/trader-app/src/zones/renderers/index.tsx`:

```tsx
case 'TABLE':
  return <TableRenderer payload={component.payload} />
```

(Plus the matching `import { TableRenderer } from './TableRenderer'`.)

### 12.6 Element catalog (only if the renderer is interactive)

If your renderer supports handles (e.g. a table with row-action buttons),
define a catalog inside its file the same way `FormRenderer` does, accept
`handles?` and `onAction?` on its Props, and apply the same derived-
interactivity rule (`handles?.length > 0 && onAction !== undefined`). See
`FormRenderer.tsx` for the canonical pattern.

If your renderer is passive (no buttons, like Markdown), accept only
`payload` — `Zone.tsx` will pass the dispatch context through
`renderZoneComponent`, but your renderer can ignore it.

### 12.7 Config author then uses it

In `render.json`:

```json
"results": {
  "templateId": "myapp-results-table",
  "title":      "Results",
  "projector":  "TABLE",
  "dataKey":    "rows"
}
```

And drops a template file `myapp-results-table.json` (any `.json` name that
isn't `workflow.json`/`render.json`/`*_jsonform.json` works; or use a custom
naming convention you wire into the loader):

```json
{
  "id": "myapp-results-table",
  "columns": [
    { "key": "sample_id", "label": "Sample ID" },
    { "key": "status",    "label": "Status" }
  ]
}
```

### 12.8 Failure modes

- **Backend emits `TABLE` but frontend doesn't know it.** Trader-app renders
  `UnknownRenderer` for that zone — a small "Unsupported zone type: TABLE"
  panel. No crash; older trader-app builds against a newer backend degrade
  visibly but safely.
- **Frontend has `TABLE` renderer but backend never emits it.** No harm —
  the renderer code is dead until something registers it.
- **Projector name collides with an existing one.** `NewAssembler` returns
  `uiprojector: duplicate projector type "TABLE"` at startup. Pick a
  different `ProjectorType` constant value.

---

## 13. Known follow-ups

These aren't bugs; they're documented because an extender will trip over
them:

- **`ZoneRendererProps<T>` is `{payload}`-only.** Renderers that want
  `handles`/`onAction` declare them inline (see `FormRenderer`'s `Props`
  type). The natural follow-up is to lift the base to `{component, onAction}`
  so every renderer has the same contract and no per-renderer prop drift
  occurs. Tracked informally; not blocked on anything. Until that lands,
  copy the FormRenderer prop pattern when you add an interactive renderer.
- **`title` is unused on the wire.** `Zone.tsx` renders the slot key, not
  the section title. Either start using it or remove the field; current
  configs include it.
- **No schema validator for `render.json`.** Authoring mistakes (typo in a
  state name, command that no section claims) surface as silent wire-shape
  drift, not validation errors. Worth adding a `make validate-configs`
  target; in the meantime, the checklist in §8 is the human substitute.