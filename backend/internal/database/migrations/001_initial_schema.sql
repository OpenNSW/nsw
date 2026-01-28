-- Migration: 001_initial_schema.sql
-- Description: Create initial database schema for NSW workflow management system
-- Created: 2026-01-24
-- Updated: 2026-01-27

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- ============================================================================
-- Table: hs_codes
-- Description: Harmonized System codes for classifying traded products
-- ============================================================================
CREATE TABLE IF NOT EXISTS hs_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hs_code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    category TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for faster lookups by code
CREATE INDEX IF NOT EXISTS idx_hs_codes_hs_code ON hs_codes(hs_code);

-- ============================================================================
-- Table: forms
-- Description: Dynamic form definitions (Schema + UI Schema)
-- ============================================================================
CREATE TABLE IF NOT EXISTS forms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    schema JSONB NOT NULL,
    ui_schema JSONB NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for active forms lookup
CREATE INDEX IF NOT EXISTS idx_forms_active ON forms(active);

-- ============================================================================
-- Table: workflow_templates
-- Description: Templates defining workflow steps and configurations
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version VARCHAR(50) NOT NULL,
    steps JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for faster lookups by version
CREATE INDEX IF NOT EXISTS idx_workflow_templates_version ON workflow_templates(version);

-- ============================================================================
-- Table: workflow_template_maps
-- Description: Mapping between (HS code, Trade flow) and workflow templates
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_template_maps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hs_code_id UUID NOT NULL,
    trade_flow VARCHAR(50) NOT NULL,
    workflow_template_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key constraints
    CONSTRAINT fk_workflow_template_maps_hs_code
        FOREIGN KEY (hs_code_id)
        REFERENCES hs_codes(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_workflow_template_maps_workflow_template
        FOREIGN KEY (workflow_template_id)
        REFERENCES workflow_templates(id)
        ON DELETE CASCADE
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_hs_code_id ON workflow_template_maps(hs_code_id);
CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_trade_flow ON workflow_template_maps(trade_flow);
CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_workflow_template_id ON workflow_template_maps(workflow_template_id);

-- Unique constraint to prevent duplicate mappings
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_template_maps_unique
    ON workflow_template_maps(hs_code_id, trade_flow);

-- ============================================================================
-- Table: consignments
-- Description: Consignment records for import/export workflows
-- ============================================================================
CREATE TABLE IF NOT EXISTS consignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_flow VARCHAR(20) NOT NULL CHECK (trade_flow IN ('IMPORT', 'EXPORT')),
    items JSONB NOT NULL,
    trader_id VARCHAR(255) NOT NULL,
    state VARCHAR(20) NOT NULL CHECK (state IN ('IN_PROGRESS', 'REQUIRES_REWORK', 'FINISHED')),
    global_context JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_consignments_trader_id ON consignments(trader_id);
CREATE INDEX IF NOT EXISTS idx_consignments_state ON consignments(state);
CREATE INDEX IF NOT EXISTS idx_consignments_trade_flow ON consignments(trade_flow);
CREATE INDEX IF NOT EXISTS idx_consignments_created_at ON consignments(created_at DESC);

-- GIN index for JSONB items column for faster JSON queries
CREATE INDEX IF NOT EXISTS idx_consignments_items ON consignments USING GIN (items);

-- ============================================================================
-- Table: tasks
-- Description: Individual task instances within consignment workflows
-- ============================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consignment_id UUID NOT NULL,
    step_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('SIMPLE_FORM', 'WAIT_FOR_EVENT')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('LOCKED', 'READY', 'IN_PROGRESS', 'COMPLETED', 'REJECTED')),
    config JSONB NOT NULL,
    depends_on JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key constraint
    CONSTRAINT fk_tasks_consignment
        FOREIGN KEY (consignment_id)
        REFERENCES consignments(id)
        ON DELETE CASCADE
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_tasks_consignment_id ON tasks(consignment_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_step_id ON tasks(step_id);
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);

-- GIN indexes for JSONB columns for faster JSON queries
CREATE INDEX IF NOT EXISTS idx_tasks_config ON tasks USING GIN (config);
CREATE INDEX IF NOT EXISTS idx_tasks_depends_on ON tasks USING GIN (depends_on);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_tasks_consignment_status ON tasks(consignment_id, status);

-- ============================================================================
-- Comments for documentation
-- ============================================================================
COMMENT ON TABLE hs_codes IS 'Harmonized System codes for classifying traded products';
COMMENT ON TABLE workflow_templates IS 'Workflow templates defining steps and configurations';
COMMENT ON TABLE workflow_template_maps IS 'Mapping between HS codes, consignment types, and workflow templates';
COMMENT ON TABLE consignments IS 'Consignment records for import/export workflows';
COMMENT ON TABLE tasks IS 'Individual task instances within consignment workflows';

COMMENT ON COLUMN consignments.items IS 'JSONB array of items in the consignment, each with hsCode, workflowTemplateId, and tasks';
COMMENT ON COLUMN consignments.state IS 'Current state: IN_PROGRESS, REQUIRES_REWORK, or FINISHED';
COMMENT ON COLUMN tasks.depends_on IS 'JSONB map of stepID to completion status (INCOMPLETE or COMPLETED)';
COMMENT ON COLUMN tasks.config IS 'JSONB configuration specific to the task type';

-- ============================================================================
-- Migration complete
-- ============================================================================
