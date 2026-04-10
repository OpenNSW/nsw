-- ============================================================================
-- Migration: 011_workflow_tem_v2.down.sql
-- Purpose: Revert the creation of v2 workflow templates and maps.
-- ============================================================================

DROP TABLE IF EXISTS workflow_template_maps_v2;
DROP TABLE IF EXISTS workflow_template_v2;
