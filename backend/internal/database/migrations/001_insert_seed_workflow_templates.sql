-- ============================================================================
-- Migration: 001_insert_seed_workflow_templates.sql
-- Purpose: Seed workflow templates composed of workflow node templates.
-- ============================================================================

-- Seed data: export workflow template definitions
INSERT INTO workflow_templates (id, name, description, version, nodes, end_node_template_id)
VALUES
    (
        'a7b8c9d0-0001-4000-c000-000000000002',
        'Fresh Coconut Export (Conditional)',
        'Workflow for exporting fresh coconut with conditional unlock configuration and end-node completion',
        'sl-export-fresh-coconut-3.0',
        '[
            "c0000003-0003-0003-0003-000000000001",
            "c0000003-0003-0003-0003-000000000008",
            "c0000003-0003-0003-0003-000000000002",
            "c0000003-0003-0003-0003-000000000003",
            "c0000003-0003-0003-0003-000000000004",
            "e1a00001-0001-4000-b000-000000000005",
            "e1a00001-0001-4000-b000-000000000007"
        ]',
        'e1a00001-0001-4000-b000-000000000005'
    );