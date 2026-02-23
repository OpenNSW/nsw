-- Migration: 011_add_unlock_configuration_down.sql
-- Description: Rollback unlock configuration changes

ALTER TABLE workflow_templates
    DROP CONSTRAINT IF EXISTS fk_workflow_templates_end_node_template;

ALTER TABLE workflow_templates
    DROP COLUMN IF EXISTS end_node_template_id;

ALTER TABLE workflow_nodes
    DROP COLUMN IF EXISTS outcome,
    DROP COLUMN IF EXISTS unlock_configuration;

ALTER TABLE workflow_node_templates
    DROP COLUMN IF EXISTS unlock_configuration;
