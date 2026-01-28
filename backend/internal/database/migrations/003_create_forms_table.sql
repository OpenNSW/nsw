-- Migration: 003_create_forms_table.sql
-- Description: Create forms table and seed initial form definitions, and update workflow templates to use Form UUIDs.
-- Created: 2026-01-28

-- ============================================================================
-- Table: forms
-- Description: Dynamic form definitions (Schema + UI Schema)
-- ============================================================================
CREATE TABLE IF NOT EXISTS forms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    schema JSONB NOT NULL,
    ui_schema JSONB NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for active forms lookup
CREATE INDEX IF NOT EXISTS idx_forms_active ON forms(active);

-- ============================================================================
-- Seed Data: Insert Forms
-- Use specific UUIDs to allow updating workflow_templates deterministically
-- ============================================================================

-- 1. Customs Declaration (cusdec_declaration)
INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Customs Declaration',
    '{"type": "object", "required": ["declarationType", "totalInvoiceValue", "totalPackages", "totalNetWeight"], "properties": {"totalPackages": {"type": "number", "title": "Total Packages", "minimum": 0}, "declarationType": {"enum": ["EX1"], "type": "string", "title": "Declaration Type"}, "totalNetWeight": {"type": "number", "title": "Total Net Weight (kg)", "minimum": 0}, "countryOfOrigin": {"type": "string", "title": "Country of Origin", "readOnly": true, "x-globalContext": "countryOfOrigin"}, "totalInvoiceValue": {"type": "string", "title": "Total Invoice Value & Currency", "description": "Example: 1,000,000 LKR"}, "countryOfDestination": {"type": "string", "title": "Country of Destination", "readOnly": true, "x-globalContext": "countryOfDestination"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Customs Declaration", "type": "Label"}, {"scope": "#/properties/declarationType", "type": "Control"}, {"scope": "#/properties/totalInvoiceValue", "type": "Control"}, {"scope": "#/properties/totalPackages", "type": "Control"}, {"type": "HorizontalLayout", "elements": [{"scope": "#/properties/countryOfOrigin", "type": "Control", "options": {"readOnly": true}}, {"scope": "#/properties/countryOfDestination", "type": "Control", "options": {"readOnly": true}}]}, {"scope": "#/properties/totalNetWeight", "type": "Control"}]}'
);

-- 2. Phytosanitary Certificate (phytosanitary_cert)
INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Phytosanitary Certificate',
    '{"type": "object", "required": ["distinguishingMarks", "disinfestationTreatment"], "properties": {"countryOfOrigin": {"type": "string", "title": "Country of Origin", "readOnly": true, "x-globalContext": "countryOfOrigin"}, "distinguishingMarks": {"type": "string", "title": "Distinguishing Marks", "example": "BWI-UK-LOT01"}, "countryOfDestination": {"type": "string", "title": "Country of Destination", "readOnly": true, "x-globalContext": "countryOfDestination"}, "disinfestationTreatment": {"type": "string", "title": "Disinfestation Treatment", "example": "Fumigation with Methyl Bromide (CH3Br) at 48g/m³ for 24 hrs"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Phytosanitary Certificate", "type": "Label"}, {"type": "HorizontalLayout", "elements": [{"scope": "#/properties/countryOfOrigin", "type": "Control", "options": {"readOnly": true}}, {"scope": "#/properties/countryOfDestination", "type": "Control", "options": {"readOnly": true}}]}, {"scope": "#/properties/distinguishingMarks", "type": "Control"}, {"scope": "#/properties/disinfestationTreatment", "type": "Control"}]}'
);

-- 3. Health Certificate (health_cert)
INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '33333333-3333-3333-3333-333333333333',
    'Health Certificate',
    '{"type": "object", "required": ["productDescription", "batchLotNumbers", "productionExpiryDates", "microbiologicalTestReportId", "processingPlantRegistrationNo"], "properties": {"consigneeName": {"type": "string", "title": "Consignee Name", "readOnly": true, "x-globalContext": "consigneeName"}, "batchLotNumbers": {"type": "string", "title": "Batch / Lot Numbers", "description": "DC-2026-JAN-05"}, "consigneeAddress": {"type": "string", "title": "Consignee Address", "readOnly": true, "x-globalContext": "consigneeAddress"}, "productDescription": {"type": "string", "title": "Product Description", "description": "Organic Desiccated Coconut (Fine Grade)"}, "productionExpiryDates": {"type": "string", "title": "Production & Expiry Dates", "description": "Example: Production: YYYY-MM-DD, Expiry: YYYY-MM-DD"}, "microbiologicalTestReportId": {"type": "string", "title": "Microbiological Test Report ID", "description": "ITI/2026/LAB-9982"}, "processingPlantRegistrationNo": {"type": "string", "title": "Processing Plant Registration No.", "description": "CDA/REG/2025/158"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Health Certificate", "type": "Label"}, {"scope": "#/properties/consigneeName", "type": "Control", "options": {"readOnly": true}}, {"scope": "#/properties/consigneeAddress", "type": "Control", "options": {"readOnly": true, "multi": true}}, {"scope": "#/properties/productDescription", "type": "Control"}, {"scope": "#/properties/batchLotNumbers", "type": "Control"}, {"scope": "#/properties/productionExpiryDates", "type": "Control"}, {"scope": "#/properties/microbiologicalTestReportId", "type": "Control"}, {"scope": "#/properties/processingPlantRegistrationNo", "type": "Control"}]}'
);

-- 4. General Information (general_info)
INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '44444444-4444-4444-4444-444444444444',
    'General Information',
    '{"type": "object", "title": "General Info", "required": ["consigneeName", "consigneeAddress", "countryOfOrigin", "countryOfDestination"], "properties": {"consigneeName": {"type": "string", "title": "Consignee Name"}, "consigneeAddress": {"type": "string", "title": "Consignee Address"}, "countryOfOrigin": {"enum": ["LK"], "type": "string", "title": "Country of Origin"}, "countryOfDestination": {"type": "string", "title": "Country of Destination"}}}',
    '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/consigneeName", "type": "Control"}, {"scope": "#/properties/consigneeAddress", "type": "Control", "options": {"multi": true}}, {"scope": "#/properties/countryOfOrigin", "type": "Control"}, {"scope": "#/properties/countryOfDestination", "type": "Control"}]}'
);

-- 5. Placeholders for missing forms (customs-declaration-import, delivery-order)
INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '55555555-5555-5555-5555-555555555555',
    'Customs Declaration (Import) - Placeholder',
    '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}',
    '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}'
);

INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '66666666-6666-6666-6666-666666666666',
    'Delivery Order - Placeholder',
    '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}',
    '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}'
);

-- ============================================================================
-- Update: Workflow Templates
-- Replace string formIds with UUIDs
-- ============================================================================

-- Template 1 (Tea Export): d299f7e7-eca3-4399-9b22-2ae1d742109d
UPDATE workflow_templates
SET steps = '[
    {"type": "SIMPLE_FORM", "config": {"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}, "stepId": "cusdec_entry", "dependsOn": []},
    {"type": "SIMPLE_FORM", "config": {"agency": "NPQS", "service": "plant-quarantine"}, "stepId": "phytosanitary_cert", "dependsOn": ["cusdec_entry"]},
    {"type": "WAIT_FOR_EVENT", "config": {"agency": "SLTB", "service": "tea-blend-sheet"}, "stepId": "tea_blend_sheet", "dependsOn": ["cusdec_entry"]},
    {"type": "WAIT_FOR_EVENT", "config": {"event": "WAIT_FOR_EVENT"}, "stepId": "final_customs_clearance", "dependsOn": ["phytosanitary_cert", "tea_blend_sheet"]}
]'::jsonb
WHERE id = 'd299f7e7-eca3-4399-9b22-2ae1d742109d';

-- Template 2 (Coconut Oil Import): eea36780-48f2-424c-9b55-0d7394e9677d
UPDATE workflow_templates
SET steps = '[
    {"type": "WAIT_FOR_EVENT", "config": {"event": "IGM_RECEIVED"}, "stepId": "manifest_submission", "dependsOn": []},
    {"type": "SIMPLE_FORM", "config": {"formId": "55555555-5555-5555-5555-555555555555"}, "stepId": "import_cusdec", "dependsOn": ["manifest_submission"]},
    {"type": "SIMPLE_FORM", "config": {"agency": "SLSI", "service": "quality-standard-verification"}, "stepId": "slsi_clearance", "dependsOn": ["import_cusdec"]},
    {"type": "SIMPLE_FORM", "config": {"agency": "MOH", "service": "health-clearance"}, "stepId": "food_control_unit", "dependsOn": ["import_cusdec"]},
    {"type": "SIMPLE_FORM", "config": {"formId": "66666666-6666-6666-6666-666666666666"}, "stepId": "gate_pass", "dependsOn": ["slsi_clearance", "food_control_unit"]}
]'::jsonb
WHERE id = 'eea36780-48f2-424c-9b55-0d7394e9677d';

-- Template 3 (Desiccated Coconut Export): 44bbe677-d327-4968-bf72-1d314246b486
UPDATE workflow_templates
SET steps = '[
    {"type": "SIMPLE_FORM", "config": {"formId": "44444444-4444-4444-4444-444444444444"}, "stepId": "general_info", "dependsOn": []},
    {"type": "SIMPLE_FORM", "config": {"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}, "stepId": "cusdec_entry", "dependsOn": ["general_info"]},
    {"type": "SIMPLE_FORM", "config": {"agency": "NPQS", "formId": "22222222-2222-2222-2222-222222222222", "service": "plant-quarantine-phytosanitary", "submissionUrl": "http://localhost:8081/api/oga/inject", "requiresOgaVerification": true}, "stepId": "phytosanitary_cert", "dependsOn": ["cusdec_entry"]},
    {"type": "SIMPLE_FORM", "config": {"agency": "EDB", "formId": "33333333-3333-3333-3333-333333333333", "service": "export-product-registration", "submissionUrl": "http://localhost:8082/api/oga/inject", "requiresOgaVerification": true}, "stepId": "health_cert", "dependsOn": ["cusdec_entry"]},
    {"type": "WAIT_FOR_EVENT", "config": {"event": "WAIT_FOR_EVENT"}, "stepId": "final_customs_clearance", "dependsOn": ["phytosanitary_cert", "health_cert", "export_docs_and_shipping_note"]}
]'::jsonb
WHERE id = '44bbe677-d327-4968-bf72-1d314246b486';
