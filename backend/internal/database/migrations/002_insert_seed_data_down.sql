-- Migration: 002_insert_seed_data_down.sql
-- Description: Rollback script - Remove all seed data
-- Created: 2026-02-05

-- Delete in reverse order of dependencies
DELETE FROM workflow_template_maps;
DELETE FROM workflow_templates;
DELETE FROM workflow_node_templates;
DELETE FROM forms;
DELETE FROM hs_codes;
