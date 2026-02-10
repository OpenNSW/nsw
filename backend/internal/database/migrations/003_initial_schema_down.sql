-- Migration: 003_initial_schema_down.sql
-- Description: Revert pre-consignment schema changes

-- Remove constraint from workflow_nodes
ALTER TABLE workflow_nodes
    DROP CONSTRAINT IF EXISTS chk_workflow_nodes_parent_exclusive;

ALTER TABLE workflow_nodes
    DROP CONSTRAINT IF EXISTS fk_workflow_nodes_pre_consignment;

-- Drop pre_consignment_id column from workflow_nodes
DROP INDEX IF EXISTS idx_workflow_nodes_pre_consignment_state;
DROP INDEX IF EXISTS idx_workflow_nodes_pre_consignment_id;
ALTER TABLE workflow_nodes
    DROP COLUMN IF EXISTS pre_consignment_id;

-- Restore NOT NULL on consignment_id
ALTER TABLE workflow_nodes
    ALTER COLUMN consignment_id SET NOT NULL;

-- Drop pre_consignment_id column from task_infos
DROP INDEX IF EXISTS idx_task_infos_pre_consignment_id;
ALTER TABLE task_infos
    DROP COLUMN IF EXISTS pre_consignment_id;

-- Restore NOT NULL on consignment_id
ALTER TABLE task_infos
    ALTER COLUMN consignment_id SET NOT NULL;

-- Drop pre_consignments table
DROP TABLE IF EXISTS pre_consignments;

-- Drop pre_consignment_templates table
DROP TABLE IF EXISTS pre_consignment_templates;
