-- Migration: 001_initial_schema_down.sql
-- Description: Rollback initial database schema
-- Created: 2026-01-23
-- Updated: 2026-01-25

-- Drop tables in reverse order of creation (respecting foreign key dependencies)
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS consignments CASCADE;
DROP TABLE IF EXISTS workflow_template_maps CASCADE;
DROP TABLE IF EXISTS workflow_templates CASCADE;
DROP TABLE IF EXISTS hs_codes CASCADE;

-- Note: We don't drop the pgcrypto extension as it might be used by other schemas
