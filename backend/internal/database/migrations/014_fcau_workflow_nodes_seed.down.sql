-- Migration: 013_fcau_workflow_nodes_seed.down.sql
-- Description: Roll back workflow node seed data.

DELETE FROM workflow_node_templates
WHERE id IN (
    'fcau:application_submission',
    'fcau:sample_drop',
    'fcau:testing_requirement',
    'fcau:lab_payment_upload',
    'fcau:lab_results_review',
    'fcau:payment',
    'fcau:certificate_issue'
); 