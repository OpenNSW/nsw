-- Migration: insert_data.sql
-- Description: Insert initial HS codes, forms and workflow templates for NSW workflow management system
-- Created: 2026-01-24
-- Updated: 2026-01-30

-- ============================================================================
-- Seed Data: Insert HS Codes
-- ============================================================================
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '90b06747-cfa7-486b-a084-eaa1fc95595e',
    '0902.10',
    'Green tea (not fermented) in immediate packings of content not exceeding ≤ 3kg',
    'Green Tea (Small)'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '3699f18c-832a-4026-ac31-3697c3a5235d',
    '0902.10.11',
    'Certified Ceylon Green tea, flavoured, ≤ 4g (Tea bags)', 
    'Green Tea'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '1589b5b1-2db3-44ef-80c1-16151bb8d5b0',
    '0902.20',
    'Green tea (not fermented) in immediate packings > 3kg',
    'Green Tea (Bulk)'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '6aa146ba-dd72-4e5e-ae27-a1cb5d69caa5',
    '0902.30',
    'Black tea (fermented) in immediate packings ≤ 3kg',
    'Black Tea (Small)'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '2e173ef8-840b-4cc5-a667-03e1d80e04b9',
    '0902.30.21',
    'Certified Ceylon Black tea, flavoured, 4g–1kg',
    'Black Tea'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '851f0de7-0693-4cc1-9d92-19c39072bb53',
    '0902.40',
    'Black tea (fermented) in immediate packings > 3kg',
    'Black Tea (Bulk)'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '51e802c1-b57e-45ac-b563-1ae0fad06db5',
    '2101.20',
    'Extracts, essences, and concentrates of tea (Instant Tea)',
    'Value Added'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    'cb34d1ac-c48f-4370-8260-a6585009ff7e',
    '2101.20.11',
    'Instant tea, certified Ceylon origin, ≤ 4g',
    'Instant Tea'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '36a58d44-8ff6-4bea-8c9b-3db84bb5a083',
    '0801.11.10',
    'Edible Copra',
    'Kernel'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '8a0783e4-82e6-488e-b96e-6140a8912f39',
    '0801.11.90',
    'Desiccated Coconut (DC)',
    'Kernel'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '4bdfb1f0-2b71-4ddc-8b99-f31c3d7660bc',
    '0801.12.00',
    'Fresh Coconut (in the inner shell)',
    'Fresh Fruit'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    'b9e48207-2573-4c9b-89f6-06d4c22422be',
    '0801.19.30',
    'King Coconut (Thambili)',
    'Fresh Fruit'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '6b567998-4a57-4132-a595-577493aefb3f',
    '1106.30.10',
    'Coconut Flour',
    'Kernel'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '653c4c8f-8c39-4aee-86f5-7f3926d0d4c2',
    '1513.11.11',
    'Virgin Coconut Oil (VCO) - In Bulk',
    'Oils'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    'bfa92119-64d3-41f4-b21c-fd0e2eb2966b',
    '1513.11.21',
    'Virgin Coconut Oil (VCO) - Not in Bulk',
    'Oils'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '5e0f2a51-8a1e-4d7d-a00b-4565e47535d2',
    '1513.19.10',
    'Coconut Oil (Refined/Not crude) - In Bulk',
    'Oils'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '4f4fac26-bf5c-42b0-9058-b17828dcba31',
    '2008.19.20',
    'Liquid Coconut Milk',
    'Edible Prep'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '1390c617-43d4-4eee-8fff-b9f10d038981',
    '2008.19.30',
    'Coconut Milk Powder',
    'Edible Prep'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    'fd5a0de1-c547-4420-94b9-942a8349a463',
    '2106.90.97',
    'Coconut Water',
    'Beverages'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '7884654e-90e0-4b7c-a963-cf6d2b5d1c16',
    '1404.90.30',
    'Coconut Shell Pieces',
    'Non-Kernel'
);
INSERT INTO hs_codes (id, hs_code, description, category) VALUES (
    '4ba1fd6b-f42f-438f-ab9f-0ee0054ee33c',
    '1404.90.50',
    'Coconut Husk Chips',
    'Non-Kernel'
);

-- ============================================================================
-- Seed Data: Insert Forms
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

INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '77777777-7777-7777-7777-777777777777',
    'SLSI Quality Standard Verification - Placeholder',
    '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}',
    '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}'
);

INSERT INTO forms (id, name, schema, ui_schema) VALUES (
    '88888888-8888-8888-8888-888888888888',
    'Food Control Unit Health Clearance - Placeholder',
    '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}',
    '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}'
);


-- ============================================================================
-- Insert Workflow Templates
-- ============================================================================
INSERT INTO workflow_templates (id, version, steps) VALUES (
    'd299f7e7-eca3-4399-9b22-2ae1d742109d',
    'sl-export-tea-packaged-1.0',
    '[{"type": "SIMPLE_FORM", "config": {"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}, "stepId": "cusdec_entry", "dependsOn": []}, {"type": "SIMPLE_FORM", "config": {"formId": "22222222-2222-2222-2222-222222222222", "agency": "NPQS", "service": "plant-quarantine"}, "stepId": "phytosanitary_cert", "dependsOn": ["cusdec_entry"]}, {"type": "WAIT_FOR_EVENT", "config": {"agency": "SLTB", "service": "tea-blend-sheet"}, "stepId": "tea_blend_sheet", "dependsOn": ["cusdec_entry"]}, {"type": "WAIT_FOR_EVENT", "config": {"event": "WAIT_FOR_EVENT"}, "stepId": "final_customs_clearance", "dependsOn": ["phytosanitary_cert", "tea_blend_sheet"]}]'
);
INSERT INTO workflow_templates (id, version, steps) VALUES (
    'eea36780-48f2-424c-9b55-0d7394e9677d',
    'sl-import-coconut-oil-1.0',
    '[{"type": "WAIT_FOR_EVENT", "config": {"event": "IGM_RECEIVED"}, "stepId": "manifest_submission", "dependsOn": []}, {"type": "SIMPLE_FORM", "config": {"formId": "55555555-5555-5555-5555-555555555555"}, "stepId": "import_cusdec", "dependsOn": ["manifest_submission"]}, {"type": "SIMPLE_FORM", "config": {"formId": "77777777-7777-7777-7777-777777777777", "agency": "SLSI", "service": "quality-standard-verification"}, "stepId": "slsi_clearance", "dependsOn": ["import_cusdec"]}, {"type": "SIMPLE_FORM", "config": {"formId": "88888888-8888-8888-8888-888888888888", "agency": "MOH", "service": "health-clearance"}, "stepId": "food_control_unit", "dependsOn": ["import_cusdec"]}, {"type": "SIMPLE_FORM", "config": {"formId": "66666666-6666-6666-6666-666666666666"}, "stepId": "gate_pass", "dependsOn": ["slsi_clearance", "food_control_unit"]}]'
);
INSERT INTO workflow_templates (id, version, steps) VALUES (
    '44bbe677-d327-4968-bf72-1d314246b486',
    'sl-export-desiccated-coconut-1.0',
    '[{"type": "SIMPLE_FORM", "config": {"formId": "44444444-4444-4444-4444-444444444444"}, "stepId": "general_info", "dependsOn": []}, {"type": "SIMPLE_FORM", "config": {"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}, "stepId": "cusdec_entry", "dependsOn": ["general_info"]}, {"type": "SIMPLE_FORM", "config": {"agency": "NPQS", "formId": "22222222-2222-2222-2222-222222222222", "service": "plant-quarantine-phytosanitary", "submissionUrl": "http://localhost:8081/api/oga/inject", "requiresOgaVerification": true}, "stepId": "phytosanitary_cert", "dependsOn": ["cusdec_entry"]}, {"type": "SIMPLE_FORM", "config": {"agency": "EDB", "formId": "33333333-3333-3333-3333-333333333333", "service": "export-product-registration", "submissionUrl": "http://localhost:8082/api/oga/inject", "requiresOgaVerification": true}, "stepId": "health_cert", "dependsOn": ["cusdec_entry"]}, {"type": "WAIT_FOR_EVENT", "config": {"event": "WAIT_FOR_EVENT", "externalServiceUrl": "http://localhost:3001/api/process-task"}, "stepId": "final_customs_clearance", "dependsOn": ["phytosanitary_cert", "health_cert"]}]'

);

-- ============================================================================
-- Insert Workflow Template Maps
-- ============================================================================
INSERT INTO workflow_template_maps (id, hs_code_id, trade_flow, workflow_template_id) VALUES (
    'b527cff7-9caf-4756-b143-ecf62dbc4236',
    '6aa146ba-dd72-4e5e-ae27-a1cb5d69caa5',
    'EXPORT',
    'd299f7e7-eca3-4399-9b22-2ae1d742109d'
);
INSERT INTO workflow_template_maps (id, hs_code_id, trade_flow, workflow_template_id) VALUES (
    '5a4c798a-d21f-47f6-a257-0c70a556da0e',
    '2e173ef8-840b-4cc5-a667-03e1d80e04b9',
    'EXPORT',
    'd299f7e7-eca3-4399-9b22-2ae1d742109d'
);
INSERT INTO workflow_template_maps (id, hs_code_id, trade_flow, workflow_template_id) VALUES (
    '74ab852b-5077-4111-abfb-bef40fb5d488', 
    '653c4c8f-8c39-4aee-86f5-7f3926d0d4c2',
    'IMPORT',
    'eea36780-48f2-424c-9b55-0d7394e9677d'
);
INSERT INTO workflow_template_maps (id, hs_code_id, trade_flow, workflow_template_id) VALUES (
    '3c2361ca-baf4-4847-82cc-b1cf6d84096f',
    'bfa92119-64d3-41f4-b21c-fd0e2eb2966b',
    'IMPORT',
    'eea36780-48f2-424c-9b55-0d7394e9677d'
);
INSERT INTO workflow_template_maps (id, hs_code_id, trade_flow, workflow_template_id) VALUES (
    '5edbd5cb-b347-4272-9541-eee10ce9c387',
    '36a58d44-8ff6-4bea-8c9b-3db84bb5a083',
    'EXPORT',
    '44bbe677-d327-4968-bf72-1d314246b486'
);
INSERT INTO workflow_template_maps (id, hs_code_id, trade_flow, workflow_template_id) VALUES (
    '688574a0-e24c-48e4-86eb-1496d5d21da2',
    '8a0783e4-82e6-488e-b96e-6140a8912f39',
    'EXPORT',
    '44bbe677-d327-4968-bf72-1d314246b486'
);