-- -- Migration: 003_insert_seed_data.sql
-- -- Description: Insert seed data for pre-consignment templates and their workflows
-- -- Created: 2026-02-09
-- -- Prerequisites: Run after 003_initial_schema.sql

-- -- ============================================================================
-- -- Seed Data: Pre-Consignment Forms
-- -- ============================================================================

-- -- -- Form 1: Business Registration
-- -- INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
-- --     'f0000002-0001-0001-0001-000000000001',
-- --     'Business Registration',
-- --     'Business registration form for traders',
-- --     '{"type": "object", "required": ["businessName", "registrationNumber"], "properties": {"businessName": {"type": "string", "title": "Business Name", "x-globalContext": {"writeTo": "roc:br:business_name"}}, "businessType": {"enum": ["Sole Proprietorship", "Partnership", "Private Limited", "Public Limited"], "type": "string", "title": "Business Type"}, "registeredAddress": {"type": "string", "title": "Registered Address"}, "registrationNumber": {"type": "string", "title": "Business Registration Number", "x-globalContext": {"writeTo": "roc:br:reg_no"}}}}',
-- --     '{"type": "VerticalLayout", "elements": [{"text": "Business Registration", "type": "Label"}, {"type": "Control", "scope": "#/properties/businessName"}, {"type": "Control", "scope": "#/properties/registrationNumber"}, {"type": "Control", "scope": "#/properties/businessType"}, {"type": "Control", "scope": "#/properties/registeredAddress"}]}',
-- --     '1.0',
-- --     true
-- -- );

-- -- -- Form 2: TIN Registration
-- -- INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
-- --     'f0000002-0001-0001-0001-000000000002',
-- --     'TIN Registration',
-- --     'Taxpayer Identification number',
-- --     '{"type": "object", "required": ["businessName", "registrationNumber", "tinNumber", "registrationDate", "tinCertificate"], "properties": {"tinNumber": {"type": "string", "title": "TIN Number"}, "businessName": {"type": "string", "title": "Business Name", "readOnly": true, "x-globalContext": {"readFrom": "roc:br:business_name"}}, "tinCertificate": {"type": "string", "title": "TIN Registration Certificate", "format": "file"}, "issuingAuthority": {"type": "string", "title": "Issuing Authority", "default": "Inland Revenue Department", "readOnly": true}, "registrationDate": {"type": "string", "title": "TIN Registration Date", "format": "date"}, "registrationNumber": {"type": "string", "title": "Business Registration Number", "readOnly": true, "x-globalContext": {"readFrom": "roc:br:reg_no"}}}}',
-- --     '{"type": "VerticalLayout", "elements": [{"text": "TIN Registration", "type": "Label"}, {"type": "Control", "scope": "#/properties/businessName"}, {"type": "Control", "scope": "#/properties/registrationNumber"}, {"type": "Control", "scope": "#/properties/tinNumber"}, {"type": "Control", "scope": "#/properties/issuingAuthority"}, {"type": "Control", "scope": "#/properties/registrationDate"}, {"type": "Control", "scope": "#/properties/tinCertificate"}]}',
-- --     '1.0',
-- --     true
-- -- );

-- -- -- Form 3: VAT Registration (CORRECTED)
-- -- INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
-- --     'f0000002-0001-0001-0001-000000000003',
-- --     'VAT Registration',
-- --     'Value Added Tax registration form',
-- --     '{"type": "object", "required": ["businessName", "registrationNumber", "vatNumber", "effectiveDate", "vatCertificate"], "properties": {"businessName": {"type": "string", "title": "Business Name", "readOnly": true, "x-globalContext": {"readFrom": "roc:br:business_name"}}, "registrationNumber": {"type": "string", "title": "Business Registration Number", "readOnly": true, "x-globalContext": {"readFrom": "roc:br:reg_no"}}, "vatNumber": {"type": "string", "title": "VAT Registration Number"}, "taxOffice": {"type": "string", "title": "Tax Office"}, "effectiveDate": {"type": "string", "title": "VAT Effective Date", "format": "date"}, "businessActivity": {"type": "string", "title": "Primary Business Activity"}, "vatCertificate": {"type": "string", "title": "VAT Registration Certificate", "format": "file"}}}',
-- --     '{"type": "VerticalLayout", "elements": [{"text": "VAT Registration", "type": "Label"}, {"type": "Control", "scope": "#/properties/businessName"}, {"type": "Control", "scope": "#/properties/registrationNumber"}, {"type": "Control", "scope": "#/properties/vatNumber"}, {"type": "Control", "scope": "#/properties/taxOffice"}, {"type": "Control", "scope": "#/properties/effectiveDate"}, {"type": "Control", "scope": "#/properties/businessActivity"}, {"type": "Control", "scope": "#/properties/vatCertificate"}]}',
-- --     '1.0',
-- --     true
-- -- );

-- -- ============================================================================
-- -- Pre-Consignment Workflow 1: Business Registration (no dependencies)
-- -- ============================================================================
-- INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES
-- ('d0000002-0001-0001-0001-000000000001', 'Business Registration Form', 'Submit business registration details', 'SIMPLE_FORM', '{"formId": "f0000002-0001-0001-0001-000000000001"}'::jsonb, '[]'::jsonb);

-- INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
--     'e0000002-0001-0001-0001-000000000001',
--     'Business Registration Workflow',
--     'Workflow for completing business registration',
--     'pre-consignment-business-registration-1.0',
--     '["d0000002-0001-0001-0001-000000000001"]'::jsonb
-- );

-- INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
--     '0c000001-0001-0001-0001-000000000002',
--     'Business Registration',
--     'Register your business with the Registrar of Companies (ROC)',
--     'e0000002-0001-0001-0001-000000000001',
--     '[]'::jsonb
-- );

-- -- ============================================================================
-- -- Pre-Consignment Workflow 2: TIN Registration (depends on Business Registration)
-- -- ============================================================================
-- INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES
-- ('d0000002-0001-0001-0001-000000000002', 'TIN Registration Form', 'Submit TIN registration details', 'SIMPLE_FORM', '{"formId": "f0000002-0001-0001-0001-000000000002"}'::jsonb, '[]'::jsonb);

-- INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
--     'e0000002-0001-0001-0001-000000000002',
--     'TIN Registration Workflow',
--     'Workflow for completing TIN registration',
--     'pre-consignment-tin-registration-1.0',
--     '["d0000002-0001-0001-0001-000000000002"]'::jsonb
-- );

-- INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
--     '0c000002-0001-0001-0001-000000000002',
--     'TIN Registration',
--     'Register for Tax Identification Number with the Inland Revenue Department',
--     'e0000002-0001-0001-0001-000000000002',
--     '["0c000001-0001-0001-0001-000000000002"]'::jsonb
-- );

-- -- ============================================================================
-- -- Pre-Consignment Workflow 3: VAT Registration (depends on Business Registration)
-- -- ============================================================================
-- INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES
-- ('d0000002-0001-0001-0001-000000000003', 'VAT Registration Form', 'Submit VAT registration details', 'SIMPLE_FORM', '{"formId": "f0000002-0001-0001-0001-000000000003"}'::jsonb, '[]'::jsonb);

-- INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
--     'e0000002-0001-0001-0001-000000000003',
--     'VAT Registration Workflow',
--     'Workflow for completing VAT registration',
--     'pre-consignment-vat-registration-1.0',
--     '["d0000002-0001-0001-0001-000000000003"]'::jsonb
-- );

-- INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
--     '0c000003-0001-0001-0001-000000000002',
--     'VAT Registration',
--     'Register for Value Added Tax with the Inland Revenue Department',
--     'e0000002-0001-0001-0001-000000000003',
--     '["0c000001-0001-0001-0001-000000000002"]'::jsonb
-- );

-- -- ============================================================================
-- -- Migration complete
-- -- ============================================================================