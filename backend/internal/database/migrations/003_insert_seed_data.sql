-- Migration: 003_insert_seed_data.sql
-- Description: Insert seed data for pre-consignment templates and their workflows
-- Created: 2026-02-09
-- Prerequisites: Run after 003_initial_schema.sql

-- ============================================================================
-- Seed Data: Pre-Consignment Forms
-- ============================================================================
-- Insert the Form Definition
INSERT INTO forms (id, name, schema, ui_schema, version, active) VALUES (
  '11111111-1111-1111-1111-111111111111', 
  'Trader Registration',
  '{"type": "object", "required": ["companyName", "businessRegNo", "tin", "vat"], "properties": {"companyName": {"type": "string", "title": "Company Name", "minLength": 3}, "businessRegNo": {"type": "string", "title": "Business Registration Number"}, "tin": {"type": "string", "title": "Taxpayer Identification Number (TIN)"}, "vat": {"type": "string", "title": "VAT Number"}, "attachment": {"type": "string", "title": "Company Profile (PDF)", "format": "data-url", "description": "Please upload your Business Registration Certificate"}}}',
  '{"type": "VerticalLayout", "elements": [{"type": "Control", "scope": "#/properties/companyName"}, {"type": "Control", "scope": "#/properties/businessRegNo"}, {"type": "Control", "scope": "#/properties/tin"}, {"type": "Control", "scope": "#/properties/vat"}, {"type": "Control", "scope": "#/properties/attachment"}]}',
  '1.0',
  true
) ON CONFLICT (id) DO UPDATE 
SET 
  schema = EXCLUDED.schema, 
  ui_schema = EXCLUDED.ui_schema,
  name = EXCLUDED.name;

-- Insert the Workflow Node Template (The "Step" definition)
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES (
  '3d662496-0182-4118-9706-5b236166113d',
  'Trader Registration Form Step',
  'Step to capture trader details',
  'SIMPLE_FORM',
  '{"formId": "11111111-1111-1111-1111-111111111111"}',
  '[]'
) ON CONFLICT (id) DO UPDATE 
SET config = EXCLUDED.config;

-- Insert the Workflow Template (The Process Definition)
INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
  'Trader Registration Workflow',
  'Workflow for onboarding new traders',
  'v1',
  '["3d662496-0182-4118-9706-5b236166113d"]'
) ON CONFLICT (id) DO NOTHING;

-- Insert the Pre-Consignment Template
INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
  '493d31d3-bfb4-489f-94a0-5a3ca2e7ca01',
  'Trader Registration',
  'Register your company to trade on the NSW Single Window',
  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
  '[]'
) ON CONFLICT (id) DO NOTHING;
