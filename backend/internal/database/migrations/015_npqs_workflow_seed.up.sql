-- ============================================================================
-- Migration: 018_npqs_workflow_seed.up.sql
-- Purpose: Seed the parent NPQS workflow definition so that consignment-based
--          startup (PUT /api/v1/consignments/{id}) maps the NPQS HS code to
--          a workflow_template_v2 row. The actual sub-workflows and task
--          template plugin properties are loaded at runtime from the JSON
--          files under backend/internal/taskv2/npqs/ (see nsw-task-flow's
--          TaskTemplateRegistry — TASK_TEMPLATES_DIR env var).
--
-- The workflow_definition mirrors backend/internal/taskv2/npqs/npqs_workflow.json
-- verbatim. Each TASK node's task_template_id refers to a SUB-WORKFLOW that
-- nsw-task-flow's TaskManager spins up on the task Temporal queue.
-- ============================================================================

-- HS code for NPQS phytosanitary export consignments
INSERT INTO hs_codes (id, hs_code, description, category)
VALUES
(
    'npqs-hs-code-0001',
    'npqs-export-phyto',
    'NPQS Export Phytosanitary Certificate — plants, produce, and plant products requiring phytosanitary inspection.',
    'NPQS'
);


-- Map the NPQS HS code (EXPORT) to the NPQS workflow template
INSERT INTO workflow_template_maps_v2 (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES
(
    'npqs-wf-map-0001',
    'npqs-hs-code-0001',
    'EXPORT',
    'npqs-export-phytosanitary-reg'
);
