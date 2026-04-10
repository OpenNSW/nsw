-- Migration: 006_insert_seed_workflow_hscode_map.down.sql
-- Description: Roll back workflow HS code mapping seed data.

DELETE FROM workflow_template_maps 
WHERE id IN ('c3d4e5f6-7890-4000-8000-000000000001');
