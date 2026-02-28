-- Migration: 015_insert_seed_data.sql
-- Description: Add seed data for the Fresh Coconut export workflow template map to show the new conditional workflow in action.
-- Created: 2026-02-28

-- ============================================================================
-- Form: Manual Inspection Form (Phytosanitary)
-- ============================================================================
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f1a00001-0001-4000-c000-000000000001',
        'Manual Inspection Form (Phytosanitary)',
        'Form for manual inspection tasks when phytosanitary certificate requires manual review',
        '{"type": "object", "properties": {"inspectionDate": {"type": "string", "format": "date", "title": "Inspection Date"}}}'::jsonb,
        '{"type": "VerticalLayout", "elements": [{"type": "Control", "scope": "#/properties/inspectionDate"}]}'::jsonb,
        1.0,
        true
       );

-- ============================================================================
-- Form: OGA response view Manual Inspection Form (Phytosanitary)
-- ============================================================================
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f1a00001-0001-4000-c000-000000000002',
        'OGA Review View (Manual Inspection)',
        'Form to render review information of phytosanitary certificate for manual inspection cases',
        '{"type": "object", "required": ["decision", "reviewedAt"], "properties": {"decision": {"enum": ["APPROVED", "REJECTED", "NEEDS_MORE_INFO"], "type": "string"}, "reviewedAt": {"type": "string", "format": "date-time"}, "reviewerNotes": {"type": "string"}}}'::jsonb,
        '{"type": "VerticalLayout", "elements": [{"type": "Control", "scope": "#/properties/decision", "options": {"format": "radio"}}, {"type": "Control", "scope": "#/properties/reviewerNotes", "options": {"multi": true}}, {"type": "Control", "scope": "#/properties/reviewedAt"}]}'::jsonb,
        1.0,
        true
       );

-- ============================================================================
-- Workflow Node Templates: Fresh Coconut Export (with UnlockConfiguration)
-- ============================================================================
-- 6-node workflow:
--   Node 1: General Information        (root, no deps)
--   Node 2: Customs Declaration        (depends on Node 1)
--   Node 3: Phytosanitary Certificate  (depends on Node 2)
--   Node 7: Manual Inspection         (depends on Node 3)
--   Node 4: Health Certificate         (depends on Node 2)
--   Node 5: Final Processing           (depends on Node 3 & 4 & 6, end node)
--
-- Unlock configurations (boolean expressions):
--   Node 1: None (root node — starts as READY)
--   Node 2: (Node1.state == "COMPLETED")
--   Node 3: (Node2.state == "COMPLETED")
--   Node 7: (Node3.state == "COMPLETED") AND (Node3.outcome == "npqs:phytosanitary:manual_review_required")'
--   Node 4: (Node2.state == "COMPLETED")
--   Node 5: ((Node7.state == "COMPLETED") OR (Node3.state == "COMPLETED" AND Node3.outcome == "npqs:phytosanitary:approved")) AND (Node4.state == "COMPLETED")

-- Node 1: General Information (root node, no dependencies)
-- Already exists from previous seed data, so we won't insert it again to avoid conflicts with existing workflow instances that reference it
-- NodeTemplateId: c0000003-0003-0003-0003-000000000001

-- Node 2: Customs Declaration (depends on Node 1)
-- Already exists from previous seed data, so we won't insert it again to avoid conflicts with existing workflow instances that reference it
-- NodeTemplateId: c0000003-0003-0003-0003-000000000002
-- Need to SET the unlock_configuration for this node to match the new conditional workflow definition
UPDATE workflow_node_templates
SET unlock_configuration = '{
  "expression": {
    "nodeTemplateId": "c0000003-0003-0003-0003-000000000001",
    "state": "COMPLETED"
  }
}'::jsonb
WHERE id = 'c0000003-0003-0003-0003-000000000002';

-- Node 3: Phytosanitary Certificate (depends on Node 2)
-- Already exists from previous seed data, so we won't insert it again to avoid conflicts with existing workflow instances that reference it
-- NodeTemplateId: c0000003-0003-0003-0003-000000000003
-- Need to SET the unlock_configuration for this node
UPDATE workflow_node_templates
SET unlock_configuration = '{
  "expression": {
    "nodeTemplateId": "c0000003-0003-0003-0003-000000000002",
    "state": "COMPLETED"
  }
}'::jsonb
WHERE id = 'c0000003-0003-0003-0003-000000000003';

-- Node 4: Health Certificate (depends on Node 2)
-- Already exists from previous seed data, so we won't insert it again to avoid conflicts with existing workflow instances that reference it
-- NodeTemplateId: c0000003-0003-0003-0003-000000000004
-- Need to SET the unlock_configuration for this node
UPDATE workflow_node_templates
SET unlock_configuration = '{
  "expression": {
    "nodeTemplateId": "c0000003-0003-0003-0003-000000000002",
    "state": "COMPLETED"
  }
}'::jsonb
WHERE id = 'c0000003-0003-0003-0003-000000000004';

-- Node 7: Manual Inspection (depends on Node 3) 
-- This is the new node we are adding for manual inspection tasks when the OGA response indicates a manual review is required. It depends on Node 3 (Phytosanitary Certificate) and will only unlock if Node 3 is completed and has the specific outcome of "npqs:phytosanitary:manual_review_required".
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on, unlock_configuration)
VALUES
    ('e1a00001-0001-4000-b000-000000000007',
     'Manual Inspection',
     'Manual inspection task for high-risk phytosanitary cases',
     'SIMPLE_FORM',
     '{"agency": "NPQS", "formId": "f1a00001-0001-4000-c000-000000000001", "service": "plant-quarantine-phytosanitary", "callback": {"response": {"display": {"formId": "f1a00001-0001-4000-c000-000000000002"}}}, "submission": {"url": "http://localhost:8081/api/oga/inject"}}'::jsonb,
     '["c0000003-0003-0003-0003-000000000003"]'::jsonb,
     '{
       "expression": {
            "allOf": [
                {
                    "nodeTemplateId": "c0000003-0003-0003-0003-000000000003",
                    "state": "COMPLETED"
                },
                {
                    "nodeTemplateId": "c0000003-0003-0003-0003-000000000003",
                    "outcome": "npqs:phytosanitary:manual_review_required"
                }
            ]
       }
     }'::jsonb);

-- Node 5: Final Processing (depends on Node 3, 4 & 7, end node)
-- Already exists from previous seed data, so we won't insert it again to avoid conflicts with existing workflow instances that reference it
-- NodeTemplateId: e1a00001-0001-4000-b000-000000000005
-- Need to SET the depends_on and unlock_configuration for this node to match the new conditional workflow definition
-- This is the final node that represents the completion of the workflow. It should only unlock when all the following conditions are met:
--   - Node 3 (Phytosanitary Certificate) is completed with outcome "npqs:phytosanitary:approved" OR Node 7 (Manual Inspection) is completed
--   - Node 4 (Health Certificate) is completed
UPDATE workflow_node_templates
SET depends_on = '["c0000003-0003-0003-0003-000000000003", "c0000003-0003-0003-0003-000000000004", "e1a00001-0001-4000-b000-000000000007"]'::jsonb,
    unlock_configuration = '{
      "expression": {
        "allOf": [
          {
            "anyOf": [
              {
                "allOf": [
                  {
                    "nodeTemplateId": "c0000003-0003-0003-0003-000000000003",
                    "state": "COMPLETED"
                  },
                  {
                    "nodeTemplateId": "c0000003-0003-0003-0003-000000000003",
                    "outcome": "npqs:phytosanitary:approved"
                  }
                ]
              },
              {
                "nodeTemplateId": "e1a00001-0001-4000-b000-000000000007",
                "state": "COMPLETED"
              }
            ]
          },
          {
            "nodeTemplateId": "c0000003-0003-0003-0003-000000000004",
            "state": "COMPLETED"
          }
        ]
      }
    }'::jsonb
WHERE id = 'e1a00001-0001-4000-b000-000000000005';

-- ============================================================================
-- Workflow Template: Fresh Coconut Export (with end_node_template_id)
-- ============================================================================
INSERT INTO workflow_templates (id, name, description, version, nodes, end_node_template_id)
VALUES ('a7b8c9d0-0001-4000-c000-000000000002',
        'Fresh Coconut Export (Conditional)',
        'Workflow for exporting fresh coconut with conditional unlock configuration and end-node completion',
        'sl-export-fresh-coconut-3.0',
        '[
          "c0000003-0003-0003-0003-000000000001",
          "c0000003-0003-0003-0003-000000000002",
          "c0000003-0003-0003-0003-000000000003",
          "c0000003-0003-0003-0003-000000000004",
          "e1a00001-0001-4000-b000-000000000005",
          "e1a00001-0001-4000-b000-000000000007"
        ]'::jsonb,
        'e1a00001-0001-4000-b000-000000000005');

-- ============================================================================
-- Workflow Template Map: Fresh Coconut (0801.12.00) → set to conditional workflow
-- ============================================================================
UPDATE workflow_template_maps
SET workflow_template_id = 'a7b8c9d0-0001-4000-c000-000000000002'
WHERE id = 'c3d4e5f6-0001-4000-d000-000000000001';
