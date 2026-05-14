-- ============================================================================
-- Migration: 015_fcau_workflow_seed.up.sql
-- Purpose: Seed the parent FCAU workflow definition so that consignment-based
--          startup (PUT /api/v1/consignments/{id}) maps the FCAU HS code to
--          a workflow_template_v2 row. The actual sub-workflows and task
--          template plugin properties are loaded at runtime from the JSON
--          files under backend/internal/template/data/fcau/ (see nsw-task-flow's
--          TaskTemplateRegistry — TASK_TEMPLATES_DIR env var).
--
-- The workflow_definition mirrors backend/internal/template/data/fcau/fcau_workflow.json
-- verbatim. Each TASK node's task_template_id refers to a SUB-WORKFLOW that
-- nsw-task-flow's TaskManager spins up on the task Temporal queue.
-- ============================================================================


-- HS code for FCAU export health certificate consignments
INSERT INTO hs_codes (id, hs_code, description, category)
VALUES
(
    'fcau-hs-code-0001',
    'fcau-export-health',
    'FCAU Export Health Certificate — processed food consignments requiring health certification.',
    'FCAU'
);


-- Map the FCAU HS code (EXPORT) to the FCAU workflow template
INSERT INTO workflow_template_maps_v2 (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES
(
    'fcau-wf-map-0001',
    'fcau-hs-code-0001',
    'EXPORT',
    'fcau-health-certificate-reg'
);
