-- ============================================================================
-- Migration: 001_initial_schema_down.sql
-- Purpose: Roll back baseline schema objects created by 001_initial_schema.sql.
-- Notes:
--   - Drops tables in reverse dependency order.
--   - Uses IF EXISTS to keep repeated rollback attempts safe.
-- ============================================================================

-- ============================================================================
-- Drop runtime and workflow instance tables
-- ============================================================================
DROP TABLE IF EXISTS user_contexts;
DROP TABLE IF EXISTS workflow_nodes;
DROP TABLE IF EXISTS pre_consignments;
DROP TABLE IF EXISTS consignments;
DROP TABLE IF EXISTS task_infos;

-- ============================================================================
-- Drop template and mapping tables
-- ============================================================================
DROP TABLE IF EXISTS pre_consignment_templates;
DROP TABLE IF EXISTS workflow_template_maps;
DROP TABLE IF EXISTS workflow_templates;
DROP TABLE IF EXISTS workflow_node_templates;

-- ============================================================================
-- Drop reference and form tables
-- ============================================================================
DROP TABLE IF EXISTS hs_codes;
DROP TABLE IF EXISTS forms;
