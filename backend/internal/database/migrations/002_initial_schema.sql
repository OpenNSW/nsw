-- Migration: 002_initial_schema.sql
-- Description: Create all core tables for NSW workflow management system
-- Created: 2026-02-05
-- Notes: Includes task_infos, forms, hs_codes, and all model workflow tables

-- ============================================================================
-- Table: task_infos
-- Description: Task executable information and state management
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_infos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    step_id VARCHAR(50) NOT NULL,
    consignment_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('SIMPLE_FORM', 'WAIT_FOR_EVENT')),
    state VARCHAR(50) NOT NULL CHECK (state IN ('IN_PROGRESS', 'COMPLETED', 'FAILED')),
    plugin_state VARCHAR(100),
    config JSONB,
    local_state JSONB,
    global_context JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for task_infos
CREATE INDEX IF NOT EXISTS idx_task_infos_consignment_id ON task_infos(consignment_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_step_id ON task_infos(step_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_status ON task_infos(state);
CREATE INDEX IF NOT EXISTS idx_task_infos_type ON task_infos(type);
CREATE INDEX IF NOT EXISTS idx_task_infos_command_set ON task_infos USING GIN (config);
CREATE INDEX IF NOT EXISTS idx_task_infos_local_state ON task_infos USING GIN (local_state);
CREATE INDEX IF NOT EXISTS idx_task_infos_global_context ON task_infos USING GIN (global_context);
CREATE INDEX IF NOT EXISTS idx_task_infos_consignment_status ON task_infos(consignment_id, state);

-- ============================================================================
-- Table: forms
-- Description: Form templates with JSON schemas and UI schemas
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

-- Index for faster lookups by name
CREATE INDEX IF NOT EXISTS idx_forms_name ON forms(name);
CREATE INDEX IF NOT EXISTS idx_forms_active ON forms(active);

-- ============================================================================
-- Table: hs_codes
-- Description: Harmonized System codes for classifying trade products
-- ============================================================================
CREATE TABLE IF NOT EXISTS hs_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hs_code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    category VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for faster lookups by hs_code
CREATE INDEX IF NOT EXISTS idx_hs_codes_hs_code ON hs_codes(hs_code);

-- ============================================================================
-- Table: workflow_templates
-- Description: Workflow templates with name, description, version, and node references
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    version VARCHAR(50) NOT NULL,
    nodes JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for workflow_templates
CREATE INDEX IF NOT EXISTS idx_workflow_templates_version ON workflow_templates(version);
CREATE INDEX IF NOT EXISTS idx_workflow_templates_name ON workflow_templates(name);
CREATE INDEX IF NOT EXISTS idx_workflow_templates_nodes ON workflow_templates USING GIN (nodes);

-- ============================================================================
-- Table: workflow_node_templates
-- Description: Templates for workflow nodes with type, config, and dependencies
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_node_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    depends_on JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for workflow_node_templates
CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_name ON workflow_node_templates(name);
CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_type ON workflow_node_templates(type);
CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_config ON workflow_node_templates USING GIN (config);
CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_depends_on ON workflow_node_templates USING GIN (depends_on);

-- ============================================================================
-- Table: workflow_template_maps
-- Description: Mapping between HS code, consignment flow, and workflow templates
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_template_maps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hs_code_id UUID NOT NULL,
    consignment_flow VARCHAR(50) NOT NULL CHECK (consignment_flow IN ('IMPORT', 'EXPORT')),
    workflow_template_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_workflow_template_maps_hs_code
        FOREIGN KEY (hs_code_id) REFERENCES hs_codes(id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_workflow_template_maps_workflow_template
        FOREIGN KEY (workflow_template_id) REFERENCES workflow_templates(id)
        ON DELETE RESTRICT ON UPDATE CASCADE
);

-- Indexes for workflow_template_maps
CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_hs_code_id ON workflow_template_maps(hs_code_id);
CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_consignment_flow ON workflow_template_maps(consignment_flow);
CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_workflow_template_id ON workflow_template_maps(workflow_template_id);

-- Unique constraint to prevent duplicate mappings
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_template_maps_unique
    ON workflow_template_maps(hs_code_id, consignment_flow);

-- ============================================================================
-- Table: consignments
-- Description: Consignment records for import/export workflows
-- ============================================================================
CREATE TABLE IF NOT EXISTS consignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flow VARCHAR(50) NOT NULL CHECK (flow IN ('IMPORT', 'EXPORT')),
    trader_id VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL CHECK (state IN ('IN_PROGRESS', 'FINISHED')),
    items JSONB NOT NULL,
    global_context JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for consignments
CREATE INDEX IF NOT EXISTS idx_consignments_trader_id ON consignments(trader_id);
CREATE INDEX IF NOT EXISTS idx_consignments_state ON consignments(state);
CREATE INDEX IF NOT EXISTS idx_consignments_flow ON consignments(flow);
CREATE INDEX IF NOT EXISTS idx_consignments_created_at ON consignments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_consignments_items ON consignments USING GIN (items);
CREATE INDEX IF NOT EXISTS idx_consignments_global_context ON consignments USING GIN (global_context);

-- ============================================================================
-- Table: workflow_nodes
-- Description: Individual workflow node instances within consignments
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consignment_id UUID NOT NULL,
    workflow_node_template_id UUID NOT NULL,
    state VARCHAR(50) NOT NULL CHECK (state IN ('LOCKED', 'READY', 'IN_PROGRESS', 'COMPLETED', 'FAILED')),
    extended_state TEXT,
    depends_on JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_workflow_nodes_consignment
        FOREIGN KEY (consignment_id) REFERENCES consignments(id)
        ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_workflow_nodes_workflow_node_template
        FOREIGN KEY (workflow_node_template_id) REFERENCES workflow_node_templates(id)
        ON DELETE RESTRICT ON UPDATE CASCADE
);

-- Indexes for workflow_nodes
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_id ON workflow_nodes(consignment_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_node_template_id ON workflow_nodes(workflow_node_template_id);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_state ON workflow_nodes(state);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_state ON workflow_nodes(consignment_id, state);
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_depends_on ON workflow_nodes USING GIN (depends_on);

-- ============================================================================
-- Comments for documentation
-- ============================================================================
COMMENT ON TABLE task_infos IS 'Task executable information and state management for the ExecutionUnit Manager';
COMMENT ON TABLE forms IS 'Form templates with JSON schemas and UI schemas';
COMMENT ON TABLE hs_codes IS 'Harmonized System codes for classifying trade products';
COMMENT ON TABLE workflow_templates IS 'Workflow templates defining the structure with name, description, version, and node references';
COMMENT ON TABLE workflow_node_templates IS 'Templates for workflow nodes with type, configuration, and dependencies';
COMMENT ON TABLE workflow_template_maps IS 'Mapping between HS codes, consignment flow, and workflow templates';
COMMENT ON TABLE consignments IS 'Consignment records for import/export workflows';
COMMENT ON TABLE workflow_nodes IS 'Individual workflow node instances within consignments';

COMMENT ON COLUMN task_infos.step_id IS 'Unique identifier of the step within the workflow template';
COMMENT ON COLUMN task_infos.plugin_state IS 'Plugin-level state for business logic';
COMMENT ON COLUMN task_infos.config IS 'JSONB configuration specific to the task type';
COMMENT ON COLUMN task_infos.local_state IS 'JSONB local state for task execution';
COMMENT ON COLUMN task_infos.global_context IS 'JSONB global context shared across task execution';
COMMENT ON COLUMN workflow_templates.nodes IS 'JSONB array of workflow node template IDs';
COMMENT ON COLUMN workflow_node_templates.name IS 'Human-readable name of the workflow node template';
COMMENT ON COLUMN workflow_node_templates.description IS 'Optional description of the workflow node template';
COMMENT ON COLUMN workflow_node_templates.type IS 'Type of the workflow node (e.g., SIMPLE_FORM, WAIT_FOR_EVENT)';
COMMENT ON COLUMN workflow_node_templates.config IS 'JSONB configuration specific to the workflow node type';
COMMENT ON COLUMN workflow_node_templates.depends_on IS 'JSONB array of workflow node template IDs this node depends on';
