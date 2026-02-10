-- Migration: 003_insert_seed_data.sql
-- Description: Insert seed data for pre-consignment templates and their workflows
-- Created: 2026-02-09
-- Prerequisites: Run after 003_initial_schema.sql

-- ============================================================================
-- Seed Data: Pre-Consignment Forms
-- ============================================================================
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    'f0000001-0001-0001-0001-000000000001',
    'Business Registration',
    'Business registration form for traders',
    '{"type": "object", "required": ["businessName", "registrationNumber", "businessType"], "properties": {"businessName": {"type": "string", "title": "Business Name"}, "registrationNumber": {"type": "string", "title": "Registration Number"}, "businessType": {"enum": ["Sole Proprietorship", "Partnership", "Private Limited", "Public Limited"], "type": "string", "title": "Business Type"}, "registeredAddress": {"type": "string", "title": "Registered Address"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Business Registration", "type": "Label"}, {"scope": "#/properties/businessName", "type": "Control"}, {"scope": "#/properties/registrationNumber", "type": "Control"}, {"scope": "#/properties/businessType", "type": "Control"}, {"scope": "#/properties/registeredAddress", "type": "Control", "options": {"multi": true}}]}',
    '1.0',
    true
);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    'f0000001-0001-0001-0001-000000000002',
    'VAT Registration',
    'Value Added Tax registration form',
    '{"type": "object", "required": ["vatNumber", "taxOffice"], "properties": {"vatNumber": {"type": "string", "title": "VAT Number"}, "taxOffice": {"type": "string", "title": "Tax Office"}, "effectiveDate": {"type": "string", "title": "Effective Date", "format": "date"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "VAT Registration", "type": "Label"}, {"scope": "#/properties/vatNumber", "type": "Control"}, {"scope": "#/properties/taxOffice", "type": "Control"}, {"scope": "#/properties/effectiveDate", "type": "Control"}]}',
    '1.0',
    true
);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    'f0000001-0001-0001-0001-000000000003',
    'TIN Registration',
    'Tax Identification Number registration form',
    '{"type": "object", "required": ["tinNumber"], "properties": {"tinNumber": {"type": "string", "title": "TIN Number"}, "issuingAuthority": {"type": "string", "title": "Issuing Authority", "default": "Inland Revenue Department"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "TIN Registration", "type": "Label"}, {"scope": "#/properties/tinNumber", "type": "Control"}, {"scope": "#/properties/issuingAuthority", "type": "Control"}]}',
    '1.0',
    true
);

-- ============================================================================
-- Pre-Consignment Workflow 1: Business Registration (no dependencies)
-- ============================================================================
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES
('d0000001-0001-0001-0001-000000000001', 'Business Registration Form', 'Submit business registration details', 'SIMPLE_FORM', '{"formId": "f0000001-0001-0001-0001-000000000001"}'::jsonb, '[]'::jsonb);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'e0000001-0001-0001-0001-000000000001',
    'Business Registration Workflow',
    'Workflow for completing business registration',
    'pre-consignment-business-registration-1.0',
    '["d0000001-0001-0001-0001-000000000001"]'::jsonb
);

INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
    '0c000001-0001-0001-0001-000000000001',
    'Business Registration',
    'Register your business with the Registrar of Companies (ROC)',
    'e0000001-0001-0001-0001-000000000001',
    '[]'::jsonb
);

-- ============================================================================
-- Pre-Consignment Workflow 2: VAT Registration (depends on Business Registration)
-- ============================================================================
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES
('d0000001-0001-0001-0001-000000000002', 'VAT Registration Form', 'Submit VAT registration details', 'SIMPLE_FORM', '{"formId": "f0000001-0001-0001-0001-000000000002"}'::jsonb, '[]'::jsonb);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'e0000001-0001-0001-0001-000000000002',
    'VAT Registration Workflow',
    'Workflow for completing VAT registration',
    'pre-consignment-vat-registration-1.0',
    '["d0000001-0001-0001-0001-000000000002"]'::jsonb
);

INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
    '0c000002-0001-0001-0001-000000000001',
    'VAT Registration',
    'Register for Value Added Tax with the Inland Revenue Department',
    'e0000001-0001-0001-0001-000000000002',
    '["0c000001-0001-0001-0001-000000000001"]'::jsonb
);

-- ============================================================================
-- Pre-Consignment Workflow 3: TIN Registration (depends on Business Registration)
-- ============================================================================
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES
('d0000001-0001-0001-0001-000000000003', 'TIN Registration Form', 'Submit TIN registration details', 'SIMPLE_FORM', '{"formId": "f0000001-0001-0001-0001-000000000003"}'::jsonb, '[]'::jsonb);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'e0000001-0001-0001-0001-000000000003',
    'TIN Registration Workflow',
    'Workflow for completing TIN registration',
    'pre-consignment-tin-registration-1.0',
    '["d0000001-0001-0001-0001-000000000003"]'::jsonb
);

INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
    '0c000003-0001-0001-0001-000000000001',
    'TIN Registration',
    'Register for Tax Identification Number with the Inland Revenue Department',
    'e0000001-0001-0001-0001-000000000003',
    '["0c000001-0001-0001-0001-000000000001"]'::jsonb
);

-- ============================================================================
-- Migration complete
-- ============================================================================
