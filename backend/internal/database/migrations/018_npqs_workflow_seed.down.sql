-- Migration: 018_npqs_workflow_seed.down.sql
-- Description: Roll back NPQS workflow template, HS code, and mapping.

DELETE FROM workflow_template_maps_v2 WHERE id = 'npqs-wf-map-0001';
DELETE FROM hs_codes               WHERE id = 'npqs-hs-code-0001';
DELETE FROM workflow_template_v2   WHERE id = 'npqs-v1';