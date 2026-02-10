-- Migration: 004_unify_task_parent_id.sql
-- Description: Replace separate consignment_id/pre_consignment_id with unified workflow_id, replace step_id with workflow_node_template_id
-- Created: 2026-02-10
-- Notes: Simplifies task model to be independent of parent record type (consignment vs pre-consignment)

-- ============================================================================
-- Alter: task_infos
-- Description: Add unified workflow_id and workflow_node_template_id columns
-- ============================================================================

-- Step 1: Add new columns as nullable initially
ALTER TABLE task_infos
    ADD COLUMN workflow_id UUID;

ALTER TABLE task_infos
    ADD COLUMN workflow_node_template_id UUID;

-- Step 2: Backfill workflow_id from consignment_id (if not null) or pre_consignment_id (if not null)
-- Assuming pre_consignment_id is new and consignment_id is being used first
UPDATE task_infos
SET workflow_id = COALESCE(consignment_id, pre_consignment_id)
WHERE workflow_id IS NULL;

-- Step 3: Backfill workflow_node_template_id - for now, use step_id parsed as UUID (if it's a valid UUID format)
-- If step_id is not a UUID format, this will need manual intervention or a different approach
-- For now, we'll assume step_id can be treated as the template identifier
-- Note: If step_id is a string that's not a UUID, you may need to map it through a lookup table
-- IMPORTANT: This currently sets the column to NULL because step_id (string) does not reliably map to template UUIDs
-- A proper backfill strategy (e.g., via lookup table or one-time script) must be executed before enforcing NOT NULL
UPDATE task_infos
SET workflow_node_template_id = NULL -- Set to NULL initially; manual mapping may be needed
WHERE workflow_node_template_id IS NULL;

-- Step 4: Make workflow_id column NOT NULL
-- NOTE: The 'workflow_node_template_id' column is intentionally left nullable.
-- A NOT NULL constraint should be added in a future migration after a proper
-- backfill strategy for existing records is executed.
ALTER TABLE task_infos
    ALTER COLUMN workflow_id SET NOT NULL;

-- Step 5: Drop old columns and constraints
-- First drop the exclusive parent constraint if it exists
ALTER TABLE task_infos
    DROP CONSTRAINT IF EXISTS chk_task_infos_parent_exclusive;

-- Drop pre_consignment_id column (added by migration 003)
ALTER TABLE task_infos
    DROP COLUMN IF EXISTS pre_consignment_id;

-- Drop consignment_id column
ALTER TABLE task_infos
    DROP COLUMN IF EXISTS consignment_id;

-- Drop step_id column
ALTER TABLE task_infos
    DROP COLUMN IF EXISTS step_id;

-- Step 6: Create index on workflow_id for lookups
CREATE INDEX IF NOT EXISTS idx_task_infos_workflow_id ON task_infos(workflow_id);

-- Step 7: Create index on workflow_node_template_id
CREATE INDEX IF NOT EXISTS idx_task_infos_workflow_node_template_id ON task_infos(workflow_node_template_id);

-- Drop old indexes that are no longer needed
DROP INDEX IF EXISTS idx_task_infos_consignment_id;
DROP INDEX IF EXISTS idx_task_infos_step_id;
DROP INDEX IF EXISTS idx_task_infos_pre_consignment_id;
DROP INDEX IF EXISTS idx_task_infos_consignment_status;

-- ============================================================================
-- Comments for documentation
-- ============================================================================
COMMENT ON COLUMN task_infos.workflow_id IS 'Unified parent workflow ID - either a consignment_id or pre_consignment_id from the workflow_nodes';
COMMENT ON COLUMN task_infos.workflow_node_template_id IS 'Reference to the workflow_node_template_id; identifies the type and configuration of this task';
