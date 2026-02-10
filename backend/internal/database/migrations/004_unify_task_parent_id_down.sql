-- Migration: 004_unify_task_parent_id_down.sql
-- Description: Rollback migration for unified workflow_id
-- Reverts the separation of workflow_id into consignment_id and pre_consignment_id

-- ============================================================================
-- Revert: task_infos
-- Description: Restore consignment_id, pre_consignment_id, and step_id columns
-- ============================================================================

-- Step 1: Add back the old columns
ALTER TABLE task_infos
    ADD COLUMN consignment_id UUID;

ALTER TABLE task_infos
    ADD COLUMN pre_consignment_id UUID;

ALTER TABLE task_infos
    ADD COLUMN step_id VARCHAR(50);

-- Step 2: Backfill old columns from new unified columns
-- This approach restores the original parent type by checking for the workflow_id's existence
-- in either the consignments or pre_consignments table.

-- Backfill consignment_id where workflow_id exists in consignments table
UPDATE task_infos
SET consignment_id = workflow_id
WHERE EXISTS (SELECT 1 FROM consignments WHERE id = task_infos.workflow_id);

-- Backfill pre_consignment_id where workflow_id exists in pre_consignments table
UPDATE task_infos
SET pre_consignment_id = workflow_id
WHERE EXISTS (SELECT 1 FROM pre_consignments WHERE id = task_infos.workflow_id);

-- Backfill step_id from the workflow_node_templates table
-- This assumes workflow_node_template_id has been properly backfilled; if not, step_id will remain NULL
UPDATE task_infos ti
SET step_id = wnt.name
FROM workflow_node_templates wnt
WHERE ti.workflow_node_template_id = wnt.id
  AND ti.step_id IS NULL;

-- Step 3: Restore the exclusive parent constraint
-- This constraint enforces that exactly one of consignment_id or pre_consignment_id is NOT NULL
ALTER TABLE task_infos
    ADD CONSTRAINT chk_task_infos_parent_exclusive
        CHECK (
            (consignment_id IS NOT NULL AND pre_consignment_id IS NULL) OR
            (consignment_id IS NULL AND pre_consignment_id IS NOT NULL)
        );

-- Note: The NOT NULL constraint on consignment_id is enforced by the CHECK constraint above.
-- An explicit ALTER COLUMN SET NOT NULL is not needed and may fail if pre_consignments were used.

-- Step 4: Create indexes on old columns
CREATE INDEX IF NOT EXISTS idx_task_infos_consignment_id ON task_infos(consignment_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_pre_consignment_id ON task_infos(pre_consignment_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_step_id ON task_infos(step_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_consignment_status ON task_infos(consignment_id, state);

-- Step 5: Drop new columns and indexes
DROP INDEX IF EXISTS idx_task_infos_workflow_id;
DROP INDEX IF EXISTS idx_task_infos_workflow_node_template_id;

ALTER TABLE task_infos
    DROP COLUMN IF EXISTS workflow_id;

ALTER TABLE task_infos
    DROP COLUMN IF EXISTS workflow_node_template_id;
