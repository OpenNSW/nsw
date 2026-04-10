-- ============================================================================
-- Migration: 001_insert_seed_workflow_hscode_map.sql
-- Purpose: Map HS codes and flow type to workflow templates.
-- ============================================================================

-- Seed data: workflow template mapping by HS code and consignment flow
INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES
    -- Mapping for Fresh Coconut Export Workflow
    (
        'c3d4e5f6-0001-4000-d000-000000000001',
        '4bdfb1f0-2b71-4ddc-8b99-f31c3d7660bc',
        'EXPORT',
        'a7b8c9d0-0001-4000-c000-000000000002'
    );