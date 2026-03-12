-- ============================================================================
-- Migration: 001_initial_schema.sql
-- Purpose: Create baseline schema objects, constraints, indexes, and metadata comments.
-- Notes:
--   - Uses IF NOT EXISTS to keep re-runs safe for table/index creation.
--   - Establishes both consignment and pre-consignment workflow structures.
-- ============================================================================

-- ============================================================================
-- Runtime task execution tables
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_infos
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	type varchar(50) NOT NULL
		CONSTRAINT task_infos_type_check
			CHECK ((type)::text = ANY ((ARRAY['SIMPLE_FORM'::character varying, 'WAIT_FOR_EVENT'::character varying])::text[])),
	state varchar(50) NOT NULL
		CONSTRAINT task_infos_state_check
			CHECK ((state)::text = ANY (ARRAY[('INITIALIZED'::character varying)::text, ('IN_PROGRESS'::character varying)::text, ('COMPLETED'::character varying)::text, ('FAILED'::character varying)::text])),
	plugin_state varchar(100),
	config jsonb,
	local_state jsonb,
	global_context jsonb,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL,
	workflow_id uuid NOT NULL,
	workflow_node_template_id uuid
);

COMMENT ON TABLE task_infos IS 'Task executable information and state management for the ExecutionUnit Manager';

COMMENT ON COLUMN task_infos.plugin_state IS 'Plugin-level state for business logic';

COMMENT ON COLUMN task_infos.config IS 'JSONB configuration specific to the task type';

COMMENT ON COLUMN task_infos.local_state IS 'JSONB local state for task execution';

COMMENT ON COLUMN task_infos.global_context IS 'JSONB global context shared across task execution';

COMMENT ON COLUMN task_infos.workflow_id IS 'Unified parent workflow ID - either a consignment_id or pre_consignment_id from the workflow_nodes';

COMMENT ON COLUMN task_infos.workflow_node_template_id IS 'Reference to the workflow_node_template_id; identifies the type and configuration of this task';

CREATE INDEX IF NOT EXISTS idx_task_infos_status
	ON task_infos (state);

CREATE INDEX IF NOT EXISTS idx_task_infos_type
	ON task_infos (type);

CREATE INDEX IF NOT EXISTS idx_task_infos_command_set
	ON task_infos USING gin (config);

CREATE INDEX IF NOT EXISTS idx_task_infos_local_state
	ON task_infos USING gin (local_state);

CREATE INDEX IF NOT EXISTS idx_task_infos_global_context
	ON task_infos USING gin (global_context);

CREATE INDEX IF NOT EXISTS idx_task_infos_workflow_id
	ON task_infos (workflow_id);

CREATE INDEX IF NOT EXISTS idx_task_infos_workflow_node_template_id
	ON task_infos (workflow_node_template_id);

-- ============================================================================
-- Dynamic form templates
-- ============================================================================
CREATE TABLE IF NOT EXISTS forms
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	name varchar(255) NOT NULL,
	description text,
	schema jsonb NOT NULL,
	ui_schema jsonb NOT NULL,
	version varchar(50) DEFAULT '1.0'::character varying NOT NULL,
	active boolean DEFAULT true NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE forms IS 'Form templates with JSON schemas and UI schemas';

CREATE INDEX IF NOT EXISTS idx_forms_name
	ON forms (name);

CREATE INDEX IF NOT EXISTS idx_forms_active
	ON forms (active);

-- ============================================================================
-- HS code reference data
-- ============================================================================
CREATE TABLE IF NOT EXISTS hs_codes
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	hs_code varchar(50) NOT NULL
		UNIQUE,
	description text NOT NULL,
	category varchar(100),
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE hs_codes IS 'Harmonized System codes for classifying trade products';

CREATE INDEX IF NOT EXISTS idx_hs_codes_hs_code
	ON hs_codes (hs_code);

-- ============================================================================
-- Workflow template definitions
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_node_templates
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	name varchar(255) NOT NULL,
	description text,
	type varchar(50) NOT NULL,
	config jsonb NOT NULL,
	depends_on jsonb DEFAULT '[]'::jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL,
	unlock_configuration jsonb
);

COMMENT ON TABLE workflow_node_templates IS 'Templates for workflow nodes with type, configuration, and dependencies';

COMMENT ON COLUMN workflow_node_templates.name IS 'Human-readable name of the workflow node template';

COMMENT ON COLUMN workflow_node_templates.description IS 'Optional description of the workflow node template';

COMMENT ON COLUMN workflow_node_templates.type IS 'Type of the workflow node (e.g., SIMPLE_FORM, WAIT_FOR_EVENT)';

COMMENT ON COLUMN workflow_node_templates.config IS 'JSONB configuration specific to the workflow node type';

COMMENT ON COLUMN workflow_node_templates.depends_on IS 'JSONB array of workflow node template IDs this node depends on';

CREATE TABLE IF NOT EXISTS workflow_templates
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	name varchar(100) NOT NULL,
	description text,
	version varchar(50) NOT NULL,
	nodes jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL,
	end_node_template_id uuid
		CONSTRAINT fk_workflow_templates_end_node_template
			references workflow_node_templates
				ON UPDATE CASCADE ON DELETE SET NULL
);

COMMENT ON TABLE workflow_templates IS 'Workflow templates defining the structure with name, description, version, and node references';

COMMENT ON COLUMN workflow_templates.nodes IS 'JSONB array of workflow node template IDs';

CREATE INDEX IF NOT EXISTS idx_workflow_templates_version
	ON workflow_templates (version);

CREATE INDEX IF NOT EXISTS idx_workflow_templates_name
	ON workflow_templates (name);

CREATE INDEX IF NOT EXISTS idx_workflow_templates_nodes
	ON workflow_templates USING gin (nodes);

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_name
	ON workflow_node_templates (name);

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_type
	ON workflow_node_templates (type);

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_config
	ON workflow_node_templates USING gin (config);

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_depends_on
	ON workflow_node_templates USING gin (depends_on);

-- ============================================================================
-- Workflow mapping rules by HS code and flow
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_template_maps
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	hs_code_id uuid NOT NULL
		CONSTRAINT fk_workflow_template_maps_hs_code
			references hs_codes
				ON UPDATE CASCADE ON DELETE RESTRICT,
	consignment_flow varchar(50) NOT NULL
		CONSTRAINT workflow_template_maps_consignment_flow_check
			CHECK ((consignment_flow)::text = ANY ((ARRAY['IMPORT'::character varying, 'EXPORT'::character varying])::text[])),
	workflow_template_id uuid NOT NULL
		CONSTRAINT fk_workflow_template_maps_workflow_template
			references workflow_templates
				ON UPDATE CASCADE ON DELETE RESTRICT,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE workflow_template_maps IS 'Mapping between HS codes, consignment flow, and workflow templates';

CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_hs_code_id
	ON workflow_template_maps (hs_code_id);

CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_consignment_flow
	ON workflow_template_maps (consignment_flow);

CREATE INDEX IF NOT EXISTS idx_workflow_template_maps_workflow_template_id
	ON workflow_template_maps (workflow_template_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_template_maps_unique
	ON workflow_template_maps (hs_code_id, consignment_flow);

-- ============================================================================
-- Customs House Agents (CHA)
-- ============================================================================
CREATE TABLE IF NOT EXISTS customs_house_agents
(
	id         uuid      DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
	name       varchar(255)                        NOT NULL,
	description text,
	email      varchar(255),
	created_at timestamptz DEFAULT now()           NOT NULL,
	updated_at timestamptz DEFAULT now()           NOT NULL
);

COMMENT ON TABLE customs_house_agents IS 'Clearing House Agents / Customs House Agents for consignment assignment';

-- ============================================================================
-- Consignment workflow instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS consignments
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	flow varchar(50) NOT NULL
		CONSTRAINT consignments_flow_check
			CHECK ((flow)::text = ANY ((ARRAY['IMPORT'::character varying, 'EXPORT'::character varying])::text[])),
	trader_id varchar(100) NOT NULL,
	state varchar(50) NOT NULL
		CONSTRAINT consignments_state_check
			CHECK ((state)::text = ANY (ARRAY['INITIALIZED'::character varying, 'IN_PROGRESS'::character varying, 'FINISHED'::character varying])),
	items jsonb NOT NULL,
	global_context jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL,
	end_node_id uuid,
	cha_id uuid REFERENCES customs_house_agents (id)
);

COMMENT ON TABLE consignments IS 'Consignment records for import/export workflows';

COMMENT ON COLUMN consignments.cha_id IS 'Assigned Customs House Agent (CHA); set at Stage 1 by Trader';

CREATE INDEX IF NOT EXISTS idx_consignments_trader_id
	ON consignments (trader_id);

CREATE INDEX IF NOT EXISTS idx_consignments_state
	ON consignments (state);

CREATE INDEX IF NOT EXISTS idx_consignments_flow
	ON consignments (flow);

CREATE INDEX IF NOT EXISTS idx_consignments_created_at
	ON consignments (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_consignments_items
	ON consignments USING gin (items);

CREATE INDEX IF NOT EXISTS idx_consignments_global_context
	ON consignments USING gin (global_context);

CREATE INDEX IF NOT EXISTS idx_consignments_cha_id
	ON consignments (cha_id);

-- ============================================================================
-- Pre-consignment template definitions
-- ============================================================================
CREATE TABLE IF NOT EXISTS pre_consignment_templates
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	name varchar(255) NOT NULL,
	description text,
	workflow_template_id uuid NOT NULL
		CONSTRAINT fk_pre_consignment_templates_workflow_template
			references workflow_templates
				ON UPDATE CASCADE ON DELETE RESTRICT,
	depends_on jsonb DEFAULT '[]'::jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE pre_consignment_templates IS 'Templates defining pre-consignment workflows that traders complete before creating consignments';

COMMENT ON COLUMN pre_consignment_templates.depends_on IS 'JSONB array of pre-consignment template IDs that must be completed before this template can be initiated';

CREATE INDEX IF NOT EXISTS idx_pre_consignment_templates_name
	ON pre_consignment_templates (name);

CREATE INDEX IF NOT EXISTS idx_pre_consignment_templates_workflow_template_id
	ON pre_consignment_templates (workflow_template_id);

CREATE INDEX IF NOT EXISTS idx_pre_consignment_templates_depends_on
	ON pre_consignment_templates USING gin (depends_on);

-- ============================================================================
-- Pre-consignment workflow instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS pre_consignments
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	trader_id varchar(255) NOT NULL,
	pre_consignment_template_id uuid NOT NULL
		CONSTRAINT fk_pre_consignments_template
			references pre_consignment_templates
				ON UPDATE CASCADE ON DELETE RESTRICT,
	state varchar(50) NOT NULL
		CONSTRAINT pre_consignments_state_check
			CHECK ((state)::text = ANY ((ARRAY['LOCKED'::character varying, 'READY'::character varying, 'IN_PROGRESS'::character varying, 'COMPLETED'::character varying])::text[])),
	trader_context jsonb DEFAULT '{}'::jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE pre_consignments IS 'Pre-consignment workflow instances created by traders';

COMMENT ON COLUMN pre_consignments.trader_id IS 'Identifier for the trader who owns this pre-consignment';

COMMENT ON COLUMN pre_consignments.trader_context IS 'JSONB context specific to the trader, accumulated during workflow execution';

-- ============================================================================
-- Workflow node instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_nodes
(
	id uuid DEFAULT gen_random_uuid() NOT NULL
		PRIMARY KEY,
	consignment_id uuid
		CONSTRAINT fk_workflow_nodes_consignment
			references consignments
				ON UPDATE CASCADE ON DELETE CASCADE,
	workflow_node_template_id uuid NOT NULL
		CONSTRAINT fk_workflow_nodes_workflow_node_template
			references workflow_node_templates
				ON UPDATE CASCADE ON DELETE RESTRICT,
	state varchar(50) NOT NULL
		CONSTRAINT workflow_nodes_state_check
			CHECK ((state)::text = ANY ((ARRAY['LOCKED'::character varying, 'READY'::character varying, 'IN_PROGRESS'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[])),
	extended_state text,
	depends_on jsonb DEFAULT '[]'::jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL,
	pre_consignment_id uuid
		CONSTRAINT fk_workflow_nodes_pre_consignment
			references pre_consignments
				ON UPDATE CASCADE ON DELETE CASCADE,
	outcome varchar(100),
	unlock_configuration jsonb,
	CONSTRAINT chk_workflow_nodes_parent_exclusive
		CHECK (((consignment_id IS NOT NULL) AND (pre_consignment_id IS NULL)) OR ((consignment_id IS NULL) AND (pre_consignment_id IS NOT NULL)))
);

COMMENT ON TABLE workflow_nodes IS 'Individual workflow node instances within consignments';

COMMENT ON COLUMN workflow_nodes.pre_consignment_id IS 'Reference to the pre-consignment this node belongs to (mutually exclusive with consignment_id)';

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_id
	ON workflow_nodes (consignment_id);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_node_template_id
	ON workflow_nodes (workflow_node_template_id);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_state
	ON workflow_nodes (state);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_state
	ON workflow_nodes (consignment_id, state);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_depends_on
	ON workflow_nodes USING gin (depends_on);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_pre_consignment_id
	ON workflow_nodes (pre_consignment_id);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_pre_consignment_state
	ON workflow_nodes (pre_consignment_id, state);

CREATE INDEX IF NOT EXISTS idx_pre_consignments_trader_id
	ON pre_consignments (trader_id);

CREATE INDEX IF NOT EXISTS idx_pre_consignments_template_id
	ON pre_consignments (pre_consignment_template_id);

CREATE INDEX IF NOT EXISTS idx_pre_consignments_state
	ON pre_consignments (state);

CREATE INDEX IF NOT EXISTS idx_pre_consignments_trader_id_state
	ON pre_consignments (trader_id, state);

-- ============================================================================
-- User context registry
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_contexts
(
	user_id varchar(100) NOT NULL
		PRIMARY KEY,
	user_context jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE user_contexts IS 'Stores user context information including metadata in JSON format. This table is used for user identification and authorization.';

COMMENT ON COLUMN user_contexts.user_id IS 'Unique user identifier (e.g., TRADER-001)';

COMMENT ON COLUMN user_contexts.user_context IS 'JSONB field containing user metadata and context information';

