BEGIN;
-- ============================================================================
-- Migration: Create workflows table and unify workflow_id on workflow_nodes
-- ============================================================================

-- Create the workflows table (generic workflow instance)
CREATE TABLE IF NOT EXISTS workflows (
    id text NOT NULL PRIMARY KEY,
    status varchar(50) NOT NULL DEFAULT 'IN_PROGRESS'
        CONSTRAINT workflows_status_check
            CHECK ((status)::text = ANY ((ARRAY['IN_PROGRESS'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[])),
    global_context jsonb NOT NULL DEFAULT '{}'::jsonb,
    end_node_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE workflows IS 'Generic workflow instances that own workflow nodes and shared context';
COMMENT ON COLUMN workflows.status IS 'Status of the workflow: IN_PROGRESS, COMPLETED, or FAILED';
COMMENT ON COLUMN workflows.global_context IS 'JSONB shared context across all workflow nodes (e.g., trader context)';
COMMENT ON COLUMN workflows.end_node_id IS 'Optional end node ID for quick completion lookup';

CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows (status);

-- ============================================================================
-- Populate workflows from existing consignments (only those with workflow nodes)
-- ============================================================================
INSERT INTO workflows (id, status, global_context, end_node_id, created_at, updated_at)
SELECT
    c.id,
    CASE
        WHEN c.state = 'FINISHED' THEN 'COMPLETED'
        ELSE 'IN_PROGRESS'
    END,
    COALESCE(c.global_context, '{}'::jsonb),
    c.end_node_id,
    c.created_at,
    c.updated_at
FROM consignments c
WHERE c.state IN ('IN_PROGRESS', 'FINISHED')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Populate workflows from existing pre-consignments (only those with workflow nodes)
-- ============================================================================
INSERT INTO workflows (id, status, global_context, end_node_id, created_at, updated_at)
SELECT
    pc.id,
    CASE
        WHEN pc.state = 'COMPLETED' THEN 'COMPLETED'
        ELSE 'IN_PROGRESS'
    END,
    COALESCE(pc.trader_context, '{}'::jsonb),
    NULL,
    pc.created_at,
    pc.updated_at
FROM pre_consignments pc
WHERE pc.state IN ('IN_PROGRESS', 'COMPLETED')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Add workflow_id to workflow_nodes and populate from existing data
-- ============================================================================
ALTER TABLE workflow_nodes ADD COLUMN workflow_id text;

UPDATE workflow_nodes SET workflow_id = COALESCE(consignment_id, pre_consignment_id);

ALTER TABLE workflow_nodes ALTER COLUMN workflow_id SET NOT NULL;

ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_workflow;
ALTER TABLE workflow_nodes ADD CONSTRAINT fk_workflow_nodes_workflow
    FOREIGN KEY (workflow_id) REFERENCES workflows(id)
    ON UPDATE CASCADE ON DELETE CASCADE;

-- ============================================================================
-- Drop old columns and constraints from workflow_nodes
-- ============================================================================
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS chk_workflow_nodes_parent_exclusive;
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_consignment;
ALTER TABLE workflow_nodes DROP CONSTRAINT IF EXISTS fk_workflow_nodes_pre_consignment;

DROP INDEX IF EXISTS idx_workflow_nodes_consignment_id;
DROP INDEX IF EXISTS idx_workflow_nodes_pre_consignment_id;
DROP INDEX IF EXISTS idx_workflow_nodes_consignment_state;
DROP INDEX IF EXISTS idx_workflow_nodes_pre_consignment_state;

ALTER TABLE workflow_nodes DROP COLUMN consignment_id;
ALTER TABLE workflow_nodes DROP COLUMN pre_consignment_id;

-- ============================================================================
-- Drop moved columns from business tables
-- ============================================================================
DROP INDEX IF EXISTS idx_consignments_global_context;
ALTER TABLE consignments DROP COLUMN IF EXISTS global_context;
ALTER TABLE consignments DROP COLUMN IF EXISTS end_node_id;

ALTER TABLE pre_consignments DROP COLUMN IF EXISTS trader_context;

-- ============================================================================
-- New indexes for workflow_nodes
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_id ON workflow_nodes (workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_id_state ON workflow_nodes (workflow_id, state);

COMMIT;
