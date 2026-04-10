BEGIN;
-- ============================================================================
-- Migration: 010_workflow_table.down.sql
-- Purpose: Revert the unification of workflow_id and restore original columns.
-- ============================================================================

-- 1. Restore columns to business tables
ALTER TABLE consignments ADD COLUMN IF NOT EXISTS global_context jsonb;
ALTER TABLE consignments ADD COLUMN IF NOT EXISTS end_node_id text;
ALTER TABLE pre_consignments ADD COLUMN IF NOT EXISTS trader_context jsonb;

-- 2. Restore global_context data from workflows back to consignments
UPDATE consignments c
SET global_context = w.global_context,
    end_node_id = w.end_node_id
FROM workflows w
WHERE c.id = w.id;

-- 3. Restore trader_context data from workflows back to pre-consignments
UPDATE pre_consignments pc
SET trader_context = w.global_context
FROM workflows w
WHERE pc.id = w.id;

-- 4. Restore columns to workflow_nodes
ALTER TABLE workflow_nodes ADD COLUMN IF NOT EXISTS consignment_id text;
ALTER TABLE workflow_nodes ADD COLUMN IF NOT EXISTS pre_consignment_id text;

-- 5. Restore data to workflow_nodes columns
-- We need to check the presence of the ID in the corresponding table
UPDATE workflow_nodes wn
SET consignment_id = CASE WHEN EXISTS (SELECT 1 FROM consignments c WHERE c.id = wn.workflow_id) THEN workflow_id ELSE NULL END,
    pre_consignment_id = CASE WHEN EXISTS (SELECT 1 FROM pre_consignments pc WHERE pc.id = wn.workflow_id) THEN workflow_id ELSE NULL END;

-- 6. Restore constraints to workflow_nodes
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_consignment;
ALTER TABLE workflow_nodes ADD CONSTRAINT fk_workflow_nodes_consignment
    FOREIGN KEY (consignment_id) REFERENCES consignments(id)
    ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_pre_consignment;
ALTER TABLE workflow_nodes ADD CONSTRAINT fk_workflow_nodes_pre_consignment
    FOREIGN KEY (pre_consignment_id) REFERENCES pre_consignments(id)
    ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS chk_workflow_nodes_parent_exclusive;
ALTER TABLE workflow_nodes ADD CONSTRAINT chk_workflow_nodes_parent_exclusive
    CHECK (((consignment_id IS NOT NULL) AND (pre_consignment_id IS NULL)) OR ((consignment_id IS NULL) AND (pre_consignment_id IS NOT NULL)));

-- 7. Cleanup the unified structure
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_workflow;
DROP INDEX IF EXISTS idx_workflow_nodes_workflow_id;
DROP INDEX IF EXISTS idx_workflow_nodes_workflow_id_state;
ALTER TABLE workflow_nodes DROP COLUMN IF EXISTS workflow_id;

-- 8. Restore old indexes
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_id ON workflow_nodes (consignment_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_pre_consignment_id ON workflow_nodes (pre_consignment_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_state ON workflow_nodes (consignment_id, state);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_pre_consignment_state ON workflow_nodes (pre_consignment_id, state);
CREATE INDEX IF NOT EXISTS idx_consignments_global_context ON consignments USING gin (global_context);

-- 9. Drop the workflows table
DROP TABLE IF EXISTS workflows;

COMMIT;
