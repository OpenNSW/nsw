-- Migration: 011_add_unlock_configuration.sql
-- Description: Add unlock configuration support for conditional node unlocking,
--              outcome sub-states for completed nodes, and end node for workflow templates.
-- Created: 2026-02-23

-- ============================================================================
-- Alter: workflow_node_templates
-- Description: Add unlock_configuration JSONB column for conditional unlock logic
-- ============================================================================
ALTER TABLE workflow_node_templates
    ADD COLUMN IF NOT EXISTS unlock_configuration JSONB;

-- ============================================================================
-- Alter: workflow_nodes
-- Description: Add outcome column for completion sub-states and
--              unlock_configuration JSONB for resolved instance-level conditions
-- ============================================================================
ALTER TABLE workflow_nodes
    ADD COLUMN IF NOT EXISTS outcome VARCHAR(100),
    ADD COLUMN IF NOT EXISTS unlock_configuration JSONB;

-- ============================================================================
-- Alter: workflow_templates
-- Description: Add end_node_template_id to specify the end node of a workflow
-- ============================================================================
ALTER TABLE workflow_templates
    ADD COLUMN IF NOT EXISTS end_node_template_id UUID;

ALTER TABLE workflow_templates
    ADD CONSTRAINT fk_workflow_templates_end_node_template
        FOREIGN KEY (end_node_template_id) REFERENCES workflow_node_templates(id)
        ON DELETE SET NULL ON UPDATE CASCADE;
