-- Migration: 014_fcau_workflow_nodes_seed.down.sql
-- Description: Roll back FCAU workflow node template seed data.

DELETE FROM workflow_node_templates
WHERE id IN (
    'fcau-apply-health-cert-flow',
    'fcau-pay-app-fee-flow',
    'fcau-sample-decision-flow',
    'fcau-wait-sample-flow',
    'fcau-sample-assessment-flow',
    'fcau-pay-lab-fee-flow',
    'fcau-lab-test-flow',
    'fcau-issue-certificate-flow'
);