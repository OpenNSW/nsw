-- Migration: 006_insert_seed_workflow_hscode_map.down.sql
-- Description: Roll back workflow HS code mapping seed data.

DELETE FROM workflow_template_maps 
WHERE id IN ('c3d4e5f6-0001-4000-d000-000000000001');
