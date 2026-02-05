-- Migration: 002_initial_schema_down.sql
-- Description: Rollback script - Drop all core tables
-- Created: 2026-02-05

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS workflow_nodes CASCADE;
DROP TABLE IF EXISTS consignments CASCADE;
DROP TABLE IF EXISTS workflow_template_maps CASCADE;
DROP TABLE IF EXISTS workflow_node_templates CASCADE;
DROP TABLE IF EXISTS workflow_templates CASCADE;
DROP TABLE IF EXISTS hs_codes CASCADE;
DROP TABLE IF EXISTS forms CASCADE;
DROP TABLE IF EXISTS task_infos CASCADE;
