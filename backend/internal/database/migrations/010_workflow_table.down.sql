BEGIN;
-- ============================================================================
-- Migration: 010_workflow_table.down.sql
-- Purpose: Revert the unification of workflow_id and restore original columns.
-- ============================================================================

-- 1. Restore columns to business tables
ALTER TABLE consignments ADD COLUMN IF NOT EXISTS global_context jsonb;
ALTER TABLE consignments ADD COLUMN IF NOT EXISTS end_node_id text;

-- 2. Restore global_context data from workflows back to consignments
UPDATE consignments c
SET global_context = w.global_context,
    end_node_id = w.end_node_id
FROM workflows w
WHERE c.id = w.id;

-- 3. Restore columns to workflow_nodes
ALTER TABLE workflow_nodes ADD COLUMN IF NOT EXISTS consignment_id text;

-- 4. Restore data to workflow_nodes columns
UPDATE workflow_nodes wn
SET consignment_id = workflow_id
WHERE EXISTS (SELECT 1 FROM consignments c WHERE c.id = wn.workflow_id);

ALTER TABLE workflow_nodes ALTER COLUMN consignment_id SET NOT NULL;

-- 5. Restore constraints to workflow_nodes
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_consignment;
ALTER TABLE workflow_nodes ADD CONSTRAINT fk_workflow_nodes_consignment
    FOREIGN KEY (consignment_id) REFERENCES consignments(id)
    ON UPDATE CASCADE ON DELETE CASCADE;

-- 6. Cleanup the unified structure
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_workflow;
DROP INDEX IF EXISTS idx_workflow_nodes_workflow_id;
DROP INDEX IF EXISTS idx_workflow_nodes_workflow_id_state;
ALTER TABLE workflow_nodes DROP COLUMN IF EXISTS workflow_id;

-- 7. Restore old indexes
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_id ON workflow_nodes (consignment_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_state ON workflow_nodes (consignment_id, state);
CREATE INDEX IF NOT EXISTS idx_consignments_global_context ON consignments USING gin (global_context);

-- 8. Drop the workflows table
DROP TABLE IF EXISTS workflows;

COMMIT;
