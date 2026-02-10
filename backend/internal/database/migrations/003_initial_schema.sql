-- Migration: 003_initial_schema.sql
-- Description: Add pre-consignment tables and support pre-consignment workflow nodes
-- Created: 2026-02-09

-- ============================================================================
-- Table: pre_consignment_templates
-- Description: Templates defining pre-consignment workflows (e.g., business registration, VAT/TIN)
-- ============================================================================
CREATE TABLE IF NOT EXISTS pre_consignment_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    workflow_template_id UUID NOT NULL,
    depends_on JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key constraints
    CONSTRAINT fk_pre_consignment_templates_workflow_template
        FOREIGN KEY (workflow_template_id) REFERENCES workflow_templates(id)
        ON DELETE RESTRICT ON UPDATE CASCADE
);

-- Indexes for pre_consignment_templates
CREATE INDEX IF NOT EXISTS idx_pre_consignment_templates_name ON pre_consignment_templates(name);
CREATE INDEX IF NOT EXISTS idx_pre_consignment_templates_workflow_template_id ON pre_consignment_templates(workflow_template_id);
CREATE INDEX IF NOT EXISTS idx_pre_consignment_templates_depends_on ON pre_consignment_templates USING GIN (depends_on);

-- ============================================================================
-- Table: pre_consignments
-- Description: Pre-consignment instances created by traders
-- ============================================================================
CREATE TABLE IF NOT EXISTS pre_consignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trader_id VARCHAR(255) NOT NULL,
    pre_consignment_template_id UUID NOT NULL,
    state VARCHAR(50) NOT NULL CHECK (state IN ('LOCKED', 'READY', 'IN_PROGRESS', 'COMPLETED')),
    trader_context JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key constraints
    CONSTRAINT fk_pre_consignments_template
        FOREIGN KEY (pre_consignment_template_id) REFERENCES pre_consignment_templates(id)
        ON DELETE RESTRICT ON UPDATE CASCADE
);

-- Indexes for pre_consignments
CREATE INDEX IF NOT EXISTS idx_pre_consignments_trader_id ON pre_consignments(trader_id);
CREATE INDEX IF NOT EXISTS idx_pre_consignments_template_id ON pre_consignments(pre_consignment_template_id);
CREATE INDEX IF NOT EXISTS idx_pre_consignments_state ON pre_consignments(state);
CREATE INDEX IF NOT EXISTS idx_pre_consignments_trader_id_state ON pre_consignments(trader_id, state);

-- ============================================================================
-- Alter: workflow_nodes
-- Description: Make consignment_id nullable and add pre_consignment_id column
-- ============================================================================
ALTER TABLE workflow_nodes
    ALTER COLUMN consignment_id DROP NOT NULL;

ALTER TABLE workflow_nodes
    ADD COLUMN pre_consignment_id UUID;

ALTER TABLE workflow_nodes
    ADD CONSTRAINT fk_workflow_nodes_pre_consignment
        FOREIGN KEY (pre_consignment_id) REFERENCES pre_consignments(id)
        ON DELETE CASCADE ON UPDATE CASCADE;

-- Ensure exactly one of consignment_id or pre_consignment_id is set
ALTER TABLE workflow_nodes
    ADD CONSTRAINT chk_workflow_nodes_parent_exclusive
        CHECK (
            (consignment_id IS NOT NULL AND pre_consignment_id IS NULL) OR
            (consignment_id IS NULL AND pre_consignment_id IS NOT NULL)
        );

-- Index for pre_consignment_id lookups
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_pre_consignment_id ON workflow_nodes(pre_consignment_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_pre_consignment_state ON workflow_nodes(pre_consignment_id, state);

-- ============================================================================
-- Alter: task_infos
-- Description: Make consignment_id nullable and add pre_consignment_id column
-- ============================================================================
ALTER TABLE task_infos
    ALTER COLUMN consignment_id DROP NOT NULL;

ALTER TABLE task_infos
    ADD COLUMN pre_consignment_id UUID;

-- Index for pre_consignment_id lookups
CREATE INDEX IF NOT EXISTS idx_task_infos_pre_consignment_id ON task_infos(pre_consignment_id);

-- Ensure exactly one of consignment_id or pre_consignment_id is set
ALTER TABLE task_infos
    ADD CONSTRAINT chk_task_infos_parent_exclusive
        CHECK (
            (consignment_id IS NOT NULL AND pre_consignment_id IS NULL) OR
            (consignment_id IS NULL AND pre_consignment_id IS NOT NULL)
        );

-- ============================================================================
-- Comments for documentation
-- ============================================================================
COMMENT ON TABLE pre_consignment_templates IS 'Templates defining pre-consignment workflows that traders complete before creating consignments';
COMMENT ON TABLE pre_consignments IS 'Pre-consignment workflow instances created by traders';
COMMENT ON COLUMN pre_consignment_templates.depends_on IS 'JSONB array of pre-consignment template IDs that must be completed before this template can be initiated';
COMMENT ON COLUMN pre_consignments.trader_id IS 'Identifier for the trader who owns this pre-consignment';
COMMENT ON COLUMN pre_consignments.trader_context IS 'JSONB context specific to the trader, accumulated during workflow execution';
COMMENT ON COLUMN workflow_nodes.pre_consignment_id IS 'Reference to the pre-consignment this node belongs to (mutually exclusive with consignment_id)';
COMMENT ON COLUMN task_infos.pre_consignment_id IS 'Reference to the pre-consignment this task belongs to (mutually exclusive with consignment_id)';
