# Guide: Adding New Forms to OGA Portals & NSW Backend

This guide outlines how to add a new form, such as a Health Certificate verification step, to the NSW and OGA portal infrastructure. 

## Overview
Adding a new verification step to an OGA (Other Government Agency) requires changes in two main places:
1. **OGA Portal Configuration**: Defining the dynamic forms for viewing submitted data and capturing the officer's decision.
2. **NSW Core Backend Configuration**: Defining the data to collect from Traders, and telling the workflow engine where to route those forms.


## Phase 1: OGA Portal Configuration

The OGA portal uses a dynamic, JSON-schema-based approach. For each formal verification process, you provide two files.

### 1. The Trader Data View Form (`<form_id>.view.json`)
Create a read-only form for the OGA officer to view the data submitted by the trader.
* **Path**: `oga/data/forms/your-agency:service:001.view.json`
* **Format**: Standard JSON Schema (`"schema"`) and UI Layout (`"uiSchema"`).

**Key fields to include**:
* Standard string or number inputs for consignment details.
* Grouping blocks in `uiSchema` to cleanly present the data.

### 2. The Officer Review Form (`<form_id>.json`)
Create an interactive form for the OGA officer to approve, reject, or request more information based on the trader data.
* **Path**: `oga/data/forms/your-agency:service:001.json`
* **Format**: Standard JSON Schema (`"schema"`) and UI Layout (`"uiSchema"`).

**Key fields to include**:
* A `decision` property containing `APPROVED`, `REJECTED`, or `FEEDBACK_REQUESTED`.
* Additional fields for `referenceNumber` or `remarks`.
* `uiSchema` rules to conditionally show elements (e.g., only show the `referenceNumber` field when the decision is `APPROVED`).


## Phase 2: NSW Backend Configuration

For the OGA forms to be useful, the Core Workflow Engine on the NSW backend needs to know what data to ask the trader for in the first place, and where to send the data once submitted.

### 1. Seed the Trader Application Form
The exact schema and UI schema you used in the view form (from Phase 1 Step 1) must be seeded into the NSW backend so traders can fill it out.

* **Path**: `backend/internal/database/migrations/001_insert_seed_form_templates.sql`
* **Action**: Append an `INSERT` block with a brand-new distinct `id` (e.g., UUID format). Copy in the `schema` and `ui\_schema` definitions directly as JSON payloads in the SQL insert block.

### 2. Configure the Workflow Node
You must add or update an execution node in the backend workflow to trigger the exact form to be routed via the OGA service.

* **Path**: `backend/internal/database/migrations/001_insert_seed_workflow_node_templates.sql`
* **Action**: Insert a node of type `SIMPLE_FORM` (or modify an existing one).
* **Key Configuration Fields**:
    * `agency`: e.g. "FCAU" or "NPQS"
    * `formId`: The new UUID you defined in Step 1.
    * `service`: Name of the service.
    * `submission.url`: The internal docker networking URL to the specific OGA (e.g., `http://oga-fcau:8082/api/oga/inject`).
    * `submission.request.meta.verificationId`: Matches the base file name of your OGA forms (e.g., `moh:fcau:health_cert:001`).
    * `callback.response.mapping`: How to map returned values back into the central global context.


## Phase 3: Restart & Test

Because the forms are loaded into memory and the database schema is built from seeds, you need to restart the containers for changes to take effect:

1. Stop any currently running instances.
2. Drop and re-run migrations for the backend database so the new templates are injected.
3. Restart the OGA service to read the new `.json` templates from the filesystem.

Using the typical local environment, you can quickly achieve this by:
```bash
./start-docker.sh --stop
./start-docker.sh
```

Log in to the **Trader Portal**, initiate the workflow mapped to your new node, and fill out the form. The system will automatically inject it into the correct OGA instance queue