# Forms

Forms are [JSON Forms](https://jsonforms.io/) definitions (`schema` + `uiSchema`) that the OGA frontend renders for two purposes:

- **View forms** — read-only renderings of the trader-submitted data shown on the review screen.
- **Review forms** — interactive forms the officer fills in to record their review action.

A form file is purpose-agnostic: the same file can be referenced as a view form by one task and a review form by another. Forms are referenced by ID from [task configs](./task-configs.md) — they are not bound to a `taskCode` themselves.

## Form Sources

OGA can resolve form IDs from two sources, selected at startup by `OGA_FORM_SOURCE`:

> The underlying loader lives in [`oga/pkg/templatesource`](../pkg/templatesource/) and is intentionally consumer-agnostic — it returns opaque JSON bytes by ID. OGA currently only uses it for forms, but a future workflow loader (or any other service) can instantiate the same package against a different directory or manifest path.


| Source | When to use |
|---|---|
| `github` (default) | Forms are versioned centrally in [`OpenNSW/one-trade-templates`](https://github.com/OpenNSW/one-trade-templates) and shared across deployments. No per-deployment forms folder required. |
| `local` | Air-gapped or development environments, or when a deployment needs to override forms with site-specific copies. Reads from `<OGA_CONFIG_DIR>/forms/*.json` (see [File Location](#file-location-local-mode) below). |

### GitHub source

When `OGA_FORM_SOURCE=github`, OGA reads `manifest.json` from the configured repo and ref at startup, then refreshes it on a background ticker. Form files themselves are fetched lazily the first time an ID is requested and cached in memory keyed by their manifest path; the cache entry is invalidated automatically when the manifest moves an ID to a different path.

Manifest URL pattern:

```
https://raw.githubusercontent.com/<OGA_FORM_GITHUB_REPO>/<OGA_FORM_GITHUB_REF>/manifest.json
```

The manifest's `byId` object maps form ID → repo-relative path, e.g.:

```json
{
  "byId": {
    "cda-apply-cert--user-form":   "templates/cda/1-application/userinput_jsonform.json",
    "cda-apply-cert--reviewer-form": "templates/cda/1-application/reviewerinput_jsonform.json"
  }
}
```

A task config references these IDs directly:

```json
{
  "taskCode": "fcau_application_review_v1",
  "forms": {
    "view":   "fcau-apply-health-cert--user-form",
    "review": "fcau-apply-health-cert--reviewer-form"
  }
}
```

**Env vars:**

| Var | Default | Notes |
|---|---|---|
| `OGA_FORM_SOURCE` | `github` | `github` or `local`. |
| `OGA_FORM_GITHUB_REPO` | `OpenNSW/one-trade-templates` | `owner/name`. |
| `OGA_FORM_GITHUB_REF` | `main` | Branch name or commit SHA. **Pin to a commit SHA in production** so deploys are reproducible and form changes are rolled out deliberately. |
| `OGA_FORM_MANIFEST_REFRESH_INTERVAL` | `5m` | Parsed by `time.ParseDuration`. Set to `0` to disable background refresh. |

**Failure modes:**

- If the manifest fetch fails at startup, OGA exits via `log.Fatalf`. Operators should see this immediately rather than serving requests against an empty form catalog.
- If a manifest *refresh* fails after startup, the previous manifest stays in use and a warning is logged.
- If a `forms.view` / `forms.review` ID is not in the manifest, or its form file 404s, OGA logs a warning and serves the application response without that form field (same behavior as a missing local form). The review API still works.

**Note on field casing:** files in `one-trade-templates` use lowercase `uischema` and include a top-level `id`. OGA forwards form bytes verbatim — it never unmarshals them — so this is a frontend-rendering concern, not a loader concern.

## File Location (local mode)

When `OGA_FORM_SOURCE=local`, form files live in `<OGA_CONFIG_DIR>/forms/` (default: `./data/forms/`). The form ID is the filename without the `.json` extension:

```
data/forms/
├── default_review.json                       # form ID: "default_review"
└── moh_fcau_health_cert_v1_review.json       # form ID: "moh_fcau_health_cert_v1_review"
```

At startup, OGA reads every `.json` file in the directory, validates that it parses as JSON, and caches the raw bytes in memory. The forms are then resolvable by ID from task configs.

## File Structure

Each form file is a top-level object with two keys: `schema` and `uiSchema`.

```json
{
  "schema": {
    "type": "object",
    "required": ["review_outcome"],
    "properties": {
      "review_outcome": {
        "type": "string",
        "title": "Review Outcome",
        "oneOf": [
          { "const": "approve", "title": "Approve" },
          { "const": "reject",  "title": "Reject" }
        ]
      },
      "rejection_reason": { "type": "string", "title": "Reason / Comments" }
    }
  },
  "uiSchema": {
    "type": "VerticalLayout",
    "elements": [
      { "type": "Control", "scope": "#/properties/review_outcome" },
      { "type": "Control", "scope": "#/properties/rejection_reason", "options": { "multi": true } }
    ]
  }
}
```

- `schema` follows standard [JSON Schema](https://json-schema.org/) and is used for both validation and field-title lookup.
- `uiSchema` follows [JSON Forms UI Schema](https://jsonforms.io/docs/uischema/) and controls layout, rules, and rendering options.

No fields are required by the OGA service itself — the form is forwarded to the frontend verbatim. Field requirements (such as `review_outcome` for status-mapping behavior) come from the task config that *references* the form, not from the form file. See [`task-configs.md`](./task-configs.md) for the contract.

## Adding a New Form

1. Create a `.json` file in `data/forms/`. The basename becomes the form ID. Use any naming convention you like; a useful one is `<taskCode>_view` or `<taskCode>_review` to make the relationship obvious.

   ```bash
   touch data/forms/moh_fcau_health_cert_v1_review.json
   ```

2. Populate it with `schema` and `uiSchema`. Validate by running `jq . data/forms/<file>.json` or pasting into any JSON Forms playground.

3. Reference it from a task config (see [`task-configs.md`](./task-configs.md)):

   ```json
   {
     "forms": { "review": "moh_fcau_health_cert_v1_review" }
   }
   ```

4. Restart the OGA service — local forms are loaded once at startup.

## Per-Deployment Forms

Only `default_review.json` ships in the repo. Agency-specific forms live outside version control and are provided per deployment by pointing `OGA_CONFIG_DIR` at a directory containing your `forms/` (and `task-configs/`) subdirs — set `OGA_FORM_SOURCE=local` to use them. In `github` mode, forms come from the central templates repo and only the `task-configs/` subdir is read from `OGA_CONFIG_DIR`.