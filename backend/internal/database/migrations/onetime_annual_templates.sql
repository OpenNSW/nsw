-- Migration: 003_insert_seed_data.sql
-- Description: Insert seed data for pre-consignment templates and their workflows
-- Created: 2026-02-09
-- Prerequisites: Run after 003_initial_schema.sql

-- Form 1: Basic Details Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    'f0000002-0001-0001-0001-000000000004',
    'Basic Details Form',
    'Basic details form for traders',
    '{"type": "object", "required": ["businessName", "bussinessAddress", "bussinessType", "phoneNumber", "email"], "properties": {"email": {"type": "string", "title": "Email Address", "format": "email", "example": "info@abcexports.com", "x-globalContext": {"writeTo": "bi:email"}}, "phoneNumber": {"type": "string", "title": "Phone Number", "example": "+94 11 2345678", "x-globalContext": {"writeTo": "bi:phoneNumber"}}, "businessName": {"type": "string", "title": "Business Name", "example": "ABC Exports Ltd", "x-globalContext": {"writeTo": "bi:businessName"}}, "bussinessType": {"enum": ["Sole Proprietorship", "Partnership", "Private Limited", "Public Limited"], "type": "string", "title": "Business Type", "example": "Private Limited", "x-globalContext": {"writeTo": "bi:businessType"}}, "bussinessAddress": {"type": "string", "title": "Business Address", "example": "123 Main Street, Colombo 01, Sri Lanka", "x-globalContext": {"writeTo": "bi:businessAddress"}}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Basic Details", "type": "Label"}, {"type": "Control", "scope": "#/properties/businessName"}, {"type": "Control", "scope": "#/properties/bussinessAddress"}, {"type": "Control", "scope": "#/properties/bussinessType"}, {"type": "Control", "scope": "#/properties/phoneNumber"}, {"type": "Control", "scope": "#/properties/email"}]}',
    '1.0',
    true
);

-- Form 2: General Trader Verification
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    'f0000002-0001-0001-0001-000000000005',
    'General Trader Verification',
    'General verification form for traders',
    '{"type": "object", "required": ["registrationNumber", "tinNumber", "tinCertificate", "vatNumber", "vatCertificate"], "properties": {"tinNumber": {"type": "string", "title": "TIN Number", "example": "TIN123456", "x-globalContext": {"writeTo": "br:tinNumber"}}, "vatNumber": {"type": "string", "title": "VAT Registration Number", "example": "VAT123456", "x-globalContext": {"writeTo": "br:vatNumber"}}, "businessName": {"type": "string", "title": "Business Name", "readOnly": true, "x-globalContext": {"readFrom": "bi:businessName"}}, "businessType": {"type": "string", "title": "Business Type", "readOnly": true, "x-globalContext": {"readFrom": "bi:businessType"}}, "tinCertificate": {"type": "string", "title": "TIN Registration Certificate", "format": "file", "example": "tin_certificate.pdf", "x-globalContext": {"writeTo": "br:tinCertificate"}}, "vatCertificate": {"type": "string", "title": "VAT Registration Certificate", "format": "file", "example": "vat_certificate.pdf", "x-globalContext": {"writeTo": "br:vatCertificate"}}, "registeredAddress": {"type": "string", "title": "Registered Address", "readOnly": true, "x-globalContext": {"readFrom": "bi:businessAddress"}}, "registrationNumber": {"type": "string", "title": "Business Registration Number", "example": "BR123456", "x-globalContext": {"writeTo": "br:registrationNumber"}}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "General Trader Verification", "type": "Label"}, {"type": "Control", "scope": "#/properties/businessName"}, {"type": "Control", "scope": "#/properties/businessType"}, {"type": "Control", "scope": "#/properties/registeredAddress"}, {"type": "Control", "scope": "#/properties/registrationNumber"}, {"type": "Control", "scope": "#/properties/tinNumber"}, {"type": "Control", "scope": "#/properties/tinCertificate"}, {"type": "Control", "scope": "#/properties/vatNumber"}, {"type": "Control", "scope": "#/properties/vatCertificate"}]}',
    '1.0',
    true
);

-- ============================================================================
-- Pre-Consignment Workflow 1: Basic Details (no dependencies)
-- ============================================================================
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES (
    'd0000002-0001-0001-0001-000000000004',
    'Basic Details Form',
    'Submit basic business details',
    'SIMPLE_FORM',
    '{"formId": "f0000002-0001-0001-0001-000000000004"}'::jsonb,
    '[]'::jsonb
);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'e0000002-0001-0001-0001-000000000004',
    'Basic Details Workflow',
    'Workflow for submitting basic business details',
    'pre-consignment-basic-details-1.0',
    '["d0000002-0001-0001-0001-000000000004"]'::jsonb
);

INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
    '0c000004-0001-0001-0001-000000000001',
    'Basic Details',
    'Provide basic details about your business',
    'e0000002-0001-0001-0001-000000000004',
    '[]'::jsonb
);

-- ============================================================================
-- Pre-Consignment Workflow 2: General Trader Verification (depends on Business Registration, TIN Registration, VAT Registration)
-- ============================================================================
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on) VALUES (
    'd0000002-0001-0001-0001-000000000005',
    'General Trader Verification Form',
    'Submit general trader verification details',
    'SIMPLE_FORM',
    '{"agency": "IRD", "formId": "f0000002-0001-0001-0001-000000000005", "service": "inland-revenue", "submissionUrl": "http://localhost:8083/api/oga/inject", "requiresOgaVerification": true}'::jsonb,
    '[]'::jsonb
);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'e0000002-0001-0001-0001-000000000005',
    'General Trader Verification Workflow',
    'Workflow for general trader verification',
    'pre-consignment-general-trader-verification-1.0',
    '["d0000002-0001-0001-0001-000000000005"]'::jsonb
);

INSERT INTO pre_consignment_templates (id, name, description, workflow_template_id, depends_on) VALUES (
    '0c000004-0001-0001-0001-000000000002',
    'General Trader Verification',
    'Verify general trader information',
    'e0000002-0001-0001-0001-000000000005',
    '["0c000004-0001-0001-0001-000000000001"]'::jsonb
);

-- ============================================================================
-- Migration complete
-- ============================================================================