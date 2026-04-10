-- Migration: 005_insert_seed_workflow_templates.down.sql
-- Description: Roll back workflow template seed data.

DELETE FROM workflow_templates 
WHERE id IN (
    'a7b8c9d0-0001-4000-c000-000000000002'
);