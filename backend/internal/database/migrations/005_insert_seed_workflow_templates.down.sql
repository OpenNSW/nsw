-- Migration: 005_insert_seed_workflow_templates.down.sql
-- Description: Roll back workflow template seed data.

DELETE FROM workflow_templates 
WHERE id IN (
    'c0000003-0003-0003-0003-000000000001',
    'e0000002-0001-0001-0001-000000000004',
    'e0000002-0001-0001-0001-000000000005'
);
