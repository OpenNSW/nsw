-- ============================================================================
-- Migration: 003_add_go_workflow_templates.sql
-- Purpose: Add support for go-workflow format and seed desiccated coconut workflow.
-- ============================================================================

-- Create table for go-workflow JSON definitions
CREATE TABLE IF NOT EXISTS go_workflow_templates (
    id uuid NOT NULL PRIMARY KEY,
    name varchar(255) NOT NULL,
    definition jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- Add column to map go-workflow templates to HS codes
ALTER TABLE workflow_template_maps ADD COLUMN IF NOT EXISTS go_workflow_template_id uuid;

-- Seed Desiccated Coconut Workflow (go-workflow format)
-- Start -> General Information (SIMPLE_FORM) -> End
INSERT INTO go_workflow_templates (id, name, definition)
VALUES (
    '8a0783e4-82e6-488e-b96e-6140a8912f39', -- Same as HS Code for convenience or unique UUID
    'Desiccated Coconut Export Workflow',
    '{
        "workflow_id": "desiccated-coconut-export",
        "name": "Desiccated Coconut Export",
        "version": 1,
        "nodes": [
            {
                "id": "start",
                "type": "INTERNAL",
                "name": "Start",
                "internal_type": "EVENT",
                "event_type": "START",
                "x": 100,
                "y": 100
            },
            {
                "id": "general-info",
                "type": "TASK",
                "name": "General Information",
                "task_id": "c0000003-0003-0003-0003-000000000001",
                "x": 300,
                "y": 100
            },
            {
                "id": "end",
                "type": "INTERNAL",
                "name": "End",
                "internal_type": "EVENT",
                "event_type": "END",
                "x": 500,
                "y": 100
            }
        ],
        "edges": [
            {
                "id": "e1",
                "source_id": "start",
                "target_id": "general-info"
            },
            {
                "id": "e2",
                "source_id": "general-info",
                "target_id": "end"
            }
        ]
     }'::jsonb
);

-- Map HS Code for Desiccated Coconut to the new go-workflow template
INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, go_workflow_template_id)
VALUES (
    gen_random_uuid(),
    '8a0783e4-82e6-488e-b96e-6140a8912f39', -- Desiccated Coconut HS Code ID
    'EXPORT',
    '8a0783e4-82e6-488e-b96e-6140a8912f39' -- The new go-workflow template ID
) ON CONFLICT (hs_code_id, consignment_flow) DO UPDATE SET go_workflow_template_id = EXCLUDED.go_workflow_template_id;
