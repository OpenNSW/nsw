-- ============================================================================
-- Migration: 014_fcau_workflow_nodes_seed.up.sql
-- Purpose: Seed WorkflowNodeTemplates for the FCAU export health certificate
--          workflow. These rows back the parent workflow's TASK nodes
--          (`task_template_id` in 015_fcau_workflow_seed) so that
--          ConsignmentService.buildConsignmentDetailDTO can resolve each
--          node's display name/description/type when returning the
--          consignment response DTO.
--
-- The actual execution of each sub-workflow + its leaf task plugins is
-- driven by the JSON files under backend/internal/template/data/fcau/ via
-- the in-memory TaskTemplateRegistry. The rows here are metadata-only.
-- ============================================================================

INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES
    (
        'fcau-apply-health-cert-flow',
        'FCAU Health Certificate Application',
        'Trader submits the FCAU health certificate application. An FCAU officer reviews and provides a decision (approve / reject / needs more info).',
        'SIMPLE_FORM',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-pay-app-fee-flow',
        'FCAU Application Fee Payment',
        'Trader pays the FCAU application processing fee. Workflow continues once payment is confirmed.',
        'PAYMENT',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-sample-decision-flow',
        'FCAU Sample Requirement Decision',
        'An FCAU officer decides whether a physical consignment sample is required for lab testing.',
        'SIMPLE_FORM',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-wait-sample-flow',
        'Wait for Consignment Sample Delivery',
        'Waits for the FCAU facility to confirm physical receipt of the consignment sample.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-sample-assessment-flow',
        'FCAU Manual Sample Assessment',
        'An FCAU officer manually assesses the consignment sample and decides whether further laboratory testing is required.',
        'SIMPLE_FORM',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-pay-lab-fee-flow',
        'FCAU Laboratory Test Fee Payment',
        'Trader pays the FCAU laboratory testing fee. Workflow continues once payment is confirmed.',
        'PAYMENT',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-lab-test-flow',
        'FCAU Laboratory Diagnostics',
        'FCAU laboratory runs chemical and microbiological diagnostics on the consignment sample and publishes the pass/fail result.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),
    (
        'fcau-issue-certificate-flow',
        'FCAU Health Certificate Issuance',
        'An FCAU officer issues the export health certificate upon successful completion of the workflow.',
        'SIMPLE_FORM',
        '{}'::jsonb,
        '[]'
    );
