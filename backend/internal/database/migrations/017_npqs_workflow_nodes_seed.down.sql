-- Migration: 017_npqs_workflow_nodes_seed.down.sql
-- Description: Roll back NPQS workflow node template seed data.

DELETE FROM workflow_node_templates
WHERE id IN (
    'npqs:application_submission',
    'npqs:sample_wait',
    'npqs:lab_wait',
    'npqs:fumigation_wait',
    'npqs:visual_decision_wait',
    'npqs:visual_result_wait',
    'npqs:shipping_docs_submission',
    'npqs:payment',
    'npqs:certificate_issue',
    'npqs:ippc_upload'
);