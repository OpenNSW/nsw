---
name: generate-render-json
description: Generate render.json for a task directory by reading its workflow.json and templates, following the specification in docs/architecture/render-json.md.
---

# Skill: Generate `render.json`

This skill guides you through generating the `render.json` file for a task workflow directory in the OpenNSW platform.

## When to Use
Use this skill when you need to author, update, or generate a `render.json` file for a new or existing task folder under `backend/configs/`.

## Prerequisites
- The task folder must contain:
  - A workflow definition file (e.g., `workflow.json` or `*_workflow.json`).
  - Template json files (e.g., `*_jsonform.json` or other template configurations defining form schemas or markdown content).

## Step-by-Step Instructions

### Step 1: Prompt the user for the task path
First, ask the user to provide the absolute or relative path to the task directory.
Example:
> Please provide the path to the task folder (e.g., `backend/configs/fcau/3-sample_decision`).

### Step 2: Read the task directory files
Once you have the task path:
1. List all files in the task directory.
2. Read the workflow file (usually `workflow.json` or `*_workflow.json`).
3. Read all other `.json` files in the directory to extract their `id` and inspect their structure. Note that:
   - JSONForms templates typically have `schema` and `uiSchema` properties.
   - Markdown templates typically have a `template` property.
   - These template files have a top-level `id` field which must match the `templateId` in your `render.json`.

### Step 3: Analyze the Workflow & Data Mapping
Analyze the workflow structure:
1. **Identify the Task ID and Type**:
   Look at the nodes in the workflow JSON (nodes of type `TASK`). Locate the `task_template_id` or workflow ID.
2. **Identify States and Actions**:
   Determine what states the task workflow moves through.
   - If the task is interactive, look at the task outputs, input mappings, and any command/actions that progress the state.
   - Common states include `PENDING_USER`, `PENDING_PAYMENT`, `UNDER_REVIEW`, `QUEUED_EXTERNALLY`, `COMPLETED`.
   - Identify the user actions / commands (like `submit`, `save_draft`, `approve`, `reject`).
3. **Identify Data Keys**:
   Look at the `output_mapping` and `input_mapping` in the workflow JSON to see which keys in the facts data store are used (e.g., `selected_method`, `payment_method`, `reviewerform`, `payment`, etc.).

### Step 4: Follow the `render.json` Specification
Create a new `render.json` (or update the existing one) adhering to the rules in `docs/architecture/render-json.md`:

#### 1. Top-Level Identity
```json
{
  "id": "<workflow-id>:render",
  "type": "<TASK_TYPE>"
}
```
*Note: `type` should reflect the category of the task (e.g., `PAYMENT`, `REVIEW`, `APPLICATION`, etc.).*

#### 2. Configure Sections (Zones)
Define each rendering region under `"sections"`. Privileged keys `instructions`, `workspace`, and `reference` will render first in that order. Other slots render after in insertion order.
For each zone, configure:
- `templateId` (string, required): The `"id"` field extracted from the corresponding template JSON file in the task directory.
- `title` (string, optional): A descriptive display title for the section.
- `projector` (string, required):
  - `FORM` for interactive or read-only JSONForms.
  - `MARKDOWN` for text/markdown content.
  - `RAW` to project data directly.
  - `PAYMENT` for payment redirect/instructions.
- `dataKey` (string, optional): The key from `facts.Data` containing values for the projector.
- `visibleWhen` (object, optional):
  - `states` (array of strings): states in which this zone should be visible.
  - `requireDataKey` (string): name of a key that must exist and be non-nil in facts.Data.
- `handles` (array of objects, optional): For interactive zones (e.g., workspace form). Each handle has:
  - `command` (string, required): the command string.
  - `label` (string, required): button label.
  - `element` (string, optional): e.g., `primary_action`, `secondary_action`, `danger_action`.

#### 3. Configure States and Legality
Define legal commands per state under `"states"`:
```json
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
```
*Ensure state names are case-insensitive uppercase snake-case. If a state has no legal actions, specify `"actions": []` or omit it.*

### Step 5: Perform Self-Validation
Validate the draft using the checklist in the guide:
1. Every section's `templateId` must match an actual template `id` from the directory files.
2. Every `handles[].command` must appear in some state's `actions[]` to be clickable, or it will be filtered out.
3. Every state's `actions[].command` must be claimed by some section's `handles[]` if it is to be visible to the user.
4. Set states correctly based on the workflow definition.

### Step 6: Write the File
Write the generated JSON to the target path: `<task-path>/render.json`.
Confirm the file is valid JSON and formatted cleanly.
