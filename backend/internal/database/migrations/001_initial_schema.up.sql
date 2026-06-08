-- ============================================================================
-- Migration: 001_initial_schema.sql
-- Purpose: Create baseline schema objects, constraints, indexes, and metadata comments.
-- Notes:
--   - Uses IF NOT EXISTS to keep re-runs safe for table/index creation.
--   - Establishes both consignment and pre-consignment workflow structures.
-- ============================================================================

-- ============================================================================
-- HS code reference data
-- ============================================================================
CREATE TABLE IF NOT EXISTS hs_codes
(
	id text NOT NULL
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
	id text NOT NULL
		PRIMARY KEY,
	name varchar(255) NOT NULL,
	description text,
	type varchar(50) NOT NULL,
	config jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE workflow_node_templates IS 'Templates for workflow nodes with type and configuration';

COMMENT ON COLUMN workflow_node_templates.name IS 'Human-readable name of the workflow node template';

COMMENT ON COLUMN workflow_node_templates.description IS 'Optional description of the workflow node template';

COMMENT ON COLUMN workflow_node_templates.type IS 'Type of the workflow node (e.g., SIMPLE_FORM, WAIT_FOR_EVENT)';

COMMENT ON COLUMN workflow_node_templates.config IS 'JSONB configuration specific to the workflow node type';

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_name
	ON workflow_node_templates (name);

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_type
	ON workflow_node_templates (type);

CREATE INDEX IF NOT EXISTS idx_workflow_node_templates_config
	ON workflow_node_templates USING gin (config);

-- ============================================================================
-- Company profile records
-- ============================================================================
CREATE TABLE IF NOT EXISTS company_records
(
	id         varchar(100)             NOT NULL PRIMARY KEY,
	name       varchar(255)             NOT NULL,
	ou_handle  varchar(255)             NOT NULL UNIQUE,
	has_cha    boolean                  NOT NULL DEFAULT false,
	data       jsonb                    NOT NULL DEFAULT '{}',
	created_at timestamp with time zone DEFAULT now(),
	updated_at timestamp with time zone DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_company_records_ou_handle ON company_records (ou_handle);

-- ============================================================================
-- Customs House Agents (CHA)
-- ============================================================================
CREATE TABLE IF NOT EXISTS customs_house_agents
(
	id         varchar(100) NOT NULL PRIMARY KEY,
	name       varchar(255) NOT NULL,
	description text,
	email      varchar(255),
	company_id varchar(100) NOT NULL REFERENCES company_records (id),
	created_at timestamptz DEFAULT now() NOT NULL,
	updated_at timestamptz DEFAULT now() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_customs_house_agents_company_id ON customs_house_agents (company_id);

COMMENT ON TABLE customs_house_agents IS 'Clearing House Agents / Customs House Agents for consignment assignment';

-- ============================================================================
-- Consignment workflow instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS consignments
(
	id text NOT NULL
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
	end_node_id text,
	trader_company_id varchar(100) NOT NULL REFERENCES company_records (id),
	cha_company_id varchar(100) NOT NULL REFERENCES company_records (id),
	cha_id varchar(100) REFERENCES customs_house_agents (id)
);

COMMENT ON TABLE consignments IS 'Consignment records for import/export workflows';

COMMENT ON COLUMN consignments.trader_company_id IS 'Company that owns the trader; resolved from the trader user''s OU at Stage 1';
COMMENT ON COLUMN consignments.cha_company_id IS 'CHA company selected by the trader at Stage 1; constrains which CHAs may pick the consignment up';
COMMENT ON COLUMN consignments.cha_id IS 'Assigned Customs House Agent (CHA); set at Stage 2 when a CHA from cha_company_id claims the consignment';

CREATE INDEX IF NOT EXISTS idx_consignments_trader_id
	ON consignments (trader_id);

CREATE INDEX IF NOT EXISTS idx_consignments_trader_company_id
	ON consignments (trader_company_id);

CREATE INDEX IF NOT EXISTS idx_consignments_cha_company_id
	ON consignments (cha_company_id);

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
-- Workflow node instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_nodes
(
	id text NOT NULL
		PRIMARY KEY,
	consignment_id text NOT NULL
		CONSTRAINT fk_workflow_nodes_consignment
			references consignments
				ON UPDATE CASCADE ON DELETE CASCADE,
	workflow_node_template_id text NOT NULL
		CONSTRAINT fk_workflow_nodes_workflow_node_template
			references workflow_node_templates
				ON UPDATE CASCADE ON DELETE RESTRICT,
	state varchar(50) NOT NULL
		CONSTRAINT workflow_nodes_state_check
			CHECK ((state)::text = ANY ((ARRAY['LOCKED'::character varying, 'READY'::character varying, 'IN_PROGRESS'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[])),
	extended_state text,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	updated_at timestamp with time zone DEFAULT now() NOT NULL,
	outcome varchar(100)
);

COMMENT ON TABLE workflow_nodes IS 'Individual workflow node instances within consignments';

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_id
	ON workflow_nodes (consignment_id);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_node_template_id
	ON workflow_nodes (workflow_node_template_id);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_state
	ON workflow_nodes (state);

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_consignment_state
	ON workflow_nodes (consignment_id, state);

-- ============================================================================
-- User records registry
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_records
(
	id varchar(100) NOT NULL
		PRIMARY KEY,
	idp_user_id varchar(255) NOT NULL UNIQUE,
	email varchar(255) NOT NULL,
	phone_number varchar(20),
	ou_id varchar(255) NOT NULL,
	ou_handle varchar(255) NOT NULL,
	data jsonb NOT NULL,
	created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_records_ou_handle ON user_records (ou_handle);

COMMENT ON TABLE user_records IS 'Stores user record information including metadata in JSON format. This table is used for user identification and authorization.';

COMMENT ON COLUMN user_records.id IS 'Unique user identifier (e.g., TRADER-001)';

COMMENT ON COLUMN user_records.idp_user_id IS 'User ID from the identity provider';

COMMENT ON COLUMN user_records.email IS 'User email from identity claims';

COMMENT ON COLUMN user_records.ou_id IS 'User organization unit ID from identity claims';

COMMENT ON COLUMN user_records.data IS 'JSONB field containing user metadata and context information';

CREATE INDEX IF NOT EXISTS idx_user_records_idp_user_id
	ON user_records (idp_user_id);