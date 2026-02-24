-- Migration: 011_insert_unlock_config_seed.sql
-- Description: Insert seed data for a workflow template that uses unlock_configuration
--              and end_node_template_id features for Fresh Coconut (0801.12.00) export.
-- Created: 2026-02-23
-- Prerequisites: Run after 011_add_unlock_configuration.sql

-- ============================================================================
-- Workflow Node Templates: Fresh Coconut Export (with UnlockConfiguration)
-- ============================================================================
-- 5-node workflow:
--   Node 1: General Information        (root, no deps)
--   Node 2: Customs Declaration        (depends on Node 1)
--   Node 3: Phytosanitary Certificate  (depends on Node 2)
--   Node 4: Health Certificate         (depends on Node 2)
--   Node 5: Final Processing           (depends on Node 3 & 4, end node)
--
-- Unlock configurations (boolean expressions):
--   Node 1: None (root node — starts as READY)
--   Node 2: (Node1.state == "COMPLETED")
--   Node 3: (Node2.state == "COMPLETED") OR (Node2.outcome == "EXPEDITED")
--   Node 4: (Node2.state == "COMPLETED" AND Node1.state == "COMPLETED")
--   Node 5: (Node3.state == "COMPLETED" AND Node4.state == "COMPLETED")
--           OR (Node2.outcome == "FAST_TRACKED")

INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on, unlock_configuration)
VALUES
    -- Node 1: General Information (root node, no dependencies)
    ('e1a00001-0001-4000-b000-000000000001',
     'General Information',
     'General consignment information form',
     'SIMPLE_FORM',
     '{"formId": "44444444-4444-4444-4444-444444444444"}'::jsonb,
     '[]'::jsonb,
     NULL),

    -- Node 2: Customs Declaration (depends on Node 1)
    --   unlock_configuration (expression): Node1.state == "COMPLETED"
    ('e1a00001-0001-4000-b000-000000000002',
     'Customs Declaration',
     'Export customs declaration form for trade goods',
     'SIMPLE_FORM',
     '{"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}'::jsonb,
     '["e1a00001-0001-4000-b000-000000000001"]'::jsonb,
     '{
       "expression": {
         "nodeTemplateId": "e1a00001-0001-4000-b000-000000000001",
         "state": "COMPLETED"
       }
     }'::jsonb),

    -- Node 3: Phytosanitary Certificate (depends on Node 2)
    --   unlock_configuration (expression):
    --     (Node2.state == "COMPLETED") OR (Node2.outcome == "EXPEDITED")
    ('e1a00001-0001-4000-b000-000000000003',
     'Phytosanitary Certificate',
     'Phytosanitary certificate for plant products export',
     'SIMPLE_FORM',
     '{"agency": "NPQS", "formId": "22222222-2222-2222-2222-222222222222", "service": "plant-quarantine-phytosanitary", "submissionUrl": "http://localhost:8081/api/oga/inject", "requiresOgaVerification": true}'::jsonb,
     '["e1a00001-0001-4000-b000-000000000002"]'::jsonb,
     '{
       "expression": {
         "anyOf": [
           {
             "nodeTemplateId": "e1a00001-0001-4000-b000-000000000002",
             "state": "COMPLETED"
           },
           {
             "nodeTemplateId": "e1a00001-0001-4000-b000-000000000002",
             "outcome": "EXPEDITED"
           }
         ]
       }
     }'::jsonb),

    -- Node 4: Health Certificate (depends on Node 2 AND Node 1)
    --   unlock_configuration (expression):
    --     (Node2.state == "COMPLETED" AND Node1.state == "COMPLETED")
    ('e1a00001-0001-4000-b000-000000000004',
     'Health Certificate',
     'Health and safety certificate for food products',
     'SIMPLE_FORM',
     '{"agency": "EDB", "formId": "33333333-3333-3333-3333-333333333333", "service": "export-product-registration", "submissionUrl": "http://localhost:8082/api/oga/inject", "requiresOgaVerification": true}'::jsonb,
     '["e1a00001-0001-4000-b000-000000000002"]'::jsonb,
     '{
       "expression": {
         "allOf": [
           {
             "nodeTemplateId": "e1a00001-0001-4000-b000-000000000002",
             "state": "COMPLETED"
           },
           {
             "nodeTemplateId": "e1a00001-0001-4000-b000-000000000001",
             "state": "COMPLETED"
           }
         ]
       }
     }'::jsonb),

    -- Node 5: Final Processing (end node)
    --   depends_on: [Node 3, Node 4] (legacy fallback references)
    --   unlock_configuration: boolean expression
    --     (Node3.state == "COMPLETED" AND Node4.state == "COMPLETED")
    --     OR (Node2.outcome == "FAST_TRACKED")
    ('e1a00001-0001-4000-b000-000000000005',
     'Final Processing',
     'Final processing step — unlocks when both certificates are completed, or customs was fast-tracked',
     'WAIT_FOR_EVENT',
     '{"event": "WAIT_FOR_EVENT"}'::jsonb,
     '["e1a00001-0001-4000-b000-000000000003", "e1a00001-0001-4000-b000-000000000004"]'::jsonb,
     '{
       "expression": {
         "anyOf": [
           {
             "allOf": [
               {
                 "nodeTemplateId": "e1a00001-0001-4000-b000-000000000003",
                 "state": "COMPLETED"
               },
               {
                 "nodeTemplateId": "e1a00001-0001-4000-b000-000000000004",
                 "state": "COMPLETED"
               }
             ]
           },
           {
             "nodeTemplateId": "e1a00001-0001-4000-b000-000000000002",
             "outcome": "FAST_TRACKED"
           }
         ]
       }
     }'::jsonb);

-- ============================================================================
-- Add a dummy end node template for the workflow to reference (since end_node_template_id is required)
-- This node won't actually be used in the workflow since the final processing node is the logical end node, but we need a valid UUID to satisfy the foreign key constraint.
-- ============================================================================
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on, unlock_configuration)
VALUES
    ('e1a00001-0001-4000-b000-000000000006',
     'End Node Placeholder',
     'Placeholder end node template to satisfy end_node_template_id requirement',
     'SIMPLE_FORM',
     '{}'::jsonb,
     '[]'::jsonb,
     NULL);

-- ============================================================================
-- Workflow Template: Fresh Coconut Export (with end_node_template_id)
-- ============================================================================
INSERT INTO workflow_templates (id, name, description, version, nodes, end_node_template_id)
VALUES ('a7b8c9d0-0001-4000-c000-000000000001',
        'Fresh Coconut Export (Conditional)',
        'Workflow for exporting fresh coconut with conditional unlock configuration and end-node completion',
        'sl-export-fresh-coconut-3.0',
        '[
          "e1a00001-0001-4000-b000-000000000001",
          "e1a00001-0001-4000-b000-000000000002",
          "e1a00001-0001-4000-b000-000000000003",
          "e1a00001-0001-4000-b000-000000000004",
          "e1a00001-0001-4000-b000-000000000005"
        ]'::jsonb,
        'e1a00001-0001-4000-b000-000000000005');

-- ============================================================================
-- Workflow Template Map: Fresh Coconut (0801.12.00) → new conditional workflow
-- ============================================================================
INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES ('c3d4e5f6-0001-4000-d000-000000000001',
        '4bdfb1f0-2b71-4ddc-8b99-f31c3d7660bc',
        'EXPORT',
        'a7b8c9d0-0001-4000-c000-000000000001');
