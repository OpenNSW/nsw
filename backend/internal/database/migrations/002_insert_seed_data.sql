-- Migration: 002_insert_seed_data.sql
-- Description: Insert all seed data for NSW workflow management system
-- Created: 2026-02-05
-- Prerequisites: Run after 002_initial_schema.sql

-- ============================================================================
-- Seed Data: HS Codes
-- ============================================================================
INSERT INTO hs_codes (id, hs_code, description, category) VALUES 
('90b06747-cfa7-486b-a084-eaa1fc95595e', '0902.10', 'Green tea (not fermented) in immediate packings of content not exceeding ≤ 3kg', 'Green Tea (Small)'),
('3699f18c-832a-4026-ac31-3697c3a5235d', '0902.10.11', 'Certified Ceylon Green tea, flavoured, ≤ 4g (Tea bags)', 'Green Tea'),
('1589b5b1-2db3-44ef-80c1-16151bb8d5b0', '0902.20', 'Green tea (not fermented) in immediate packings > 3kg', 'Green Tea (Bulk)'),
('6aa146ba-dd72-4e5e-ae27-a1cb5d69caa5', '0902.30', 'Black tea (fermented) in immediate packings ≤ 3kg', 'Black Tea (Small)'),
('2e173ef8-840b-4cc5-a667-03e1d80e04b9', '0902.30.21', 'Certified Ceylon Black tea, flavoured, 4g–1kg', 'Black Tea'),
('851f0de7-0693-4cc1-9d92-19c39072bb53', '0902.40', 'Black tea (fermented) in immediate packings > 3kg', 'Black Tea (Bulk)'),
('51e802c1-b57e-45ac-b563-1ae0fad06db5', '2101.20', 'Extracts, essences, and concentrates of tea (Instant Tea)', 'Value Added'),
('cb34d1ac-c48f-4370-8260-a6585009ff7e', '2101.20.11', 'Instant tea, certified Ceylon origin, ≤ 4g', 'Instant Tea'),
('36a58d44-8ff6-4bea-8c9b-3db84bb5a083', '0801.11.10', 'Edible Copra', 'Kernel'),
('8a0783e4-82e6-488e-b96e-6140a8912f39', '0801.11.90', 'Desiccated Coconut (DC)', 'Kernel'),
('4bdfb1f0-2b71-4ddc-8b99-f31c3d7660bc', '0801.12.00', 'Fresh Coconut (in the inner shell)', 'Fresh Fruit'),
('b9e48207-2573-4c9b-89f6-06d4c22422be', '0801.19.30', 'King Coconut (Thambili)', 'Fresh Fruit'),
('6b567998-4a57-4132-a595-577493aefb3f', '1106.30.10', 'Coconut Flour', 'Kernel'),
('653c4c8f-8c39-4aee-86f5-7f3926d0d4c2', '1513.11.11', 'Virgin Coconut Oil (VCO) - In Bulk', 'Oils'),
('bfa92119-64d3-41f4-b21c-fd0e2eb2966b', '1513.11.21', 'Virgin Coconut Oil (VCO) - Not in Bulk', 'Oils'),
('5e0f2a51-8a1e-4d7d-a00b-4565e47535d2', '1513.19.10', 'Coconut Oil (Refined/Not crude) - In Bulk', 'Oils'),
('4f4fac26-bf5c-42b0-9058-b17828dcba31', '2008.19.20', 'Liquid Coconut Milk', 'Edible Prep'),
('1390c617-43d4-4eee-8fff-b9f10d038981', '2008.19.30', 'Coconut Milk Powder', 'Edible Prep'),
('fd5a0de1-c547-4420-94b9-942a8349a463', '2106.90.97', 'Coconut Water', 'Beverages'),
('7884654e-90e0-4b7c-a963-cf6d2b5d1c16', '1404.90.30', 'Coconut Shell Pieces', 'Non-Kernel'),
('4ba1fd6b-f42f-438f-ab9f-0ee0054ee33c', '1404.90.50', 'Coconut Husk Chips', 'Non-Kernel');

-- ============================================================================
-- Seed Data: Forms
-- ============================================================================

-- Customs Declaration
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Customs Declaration',
    'Export customs declaration form for trade goods',
    '{"type": "object", "required": ["declarationType", "totalInvoiceValue", "totalPackages", "totalNetWeight"], "properties": {"totalPackages": {"type": "number", "title": "Total Packages", "minimum": 0}, "declarationType": {"enum": ["EX1"], "type": "string", "title": "Declaration Type"}, "totalNetWeight": {"type": "number", "title": "Total Net Weight (kg)", "minimum": 0}, "countryOfOrigin": {"type": "string", "title": "Country of Origin", "readOnly": true, "x-globalContext": "countryOfOrigin"}, "totalInvoiceValue": {"type": "string", "title": "Total Invoice Value & Currency", "description": "Example: 1,000,000 LKR"}, "countryOfDestination": {"type": "string", "title": "Country of Destination", "readOnly": true, "x-globalContext": "countryOfDestination"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Customs Declaration", "type": "Label"}, {"scope": "#/properties/declarationType", "type": "Control"}, {"scope": "#/properties/totalInvoiceValue", "type": "Control"}, {"scope": "#/properties/totalPackages", "type": "Control"}, {"type": "HorizontalLayout", "elements": [{"scope": "#/properties/countryOfOrigin", "type": "Control", "options": {"readOnly": true}}, {"scope": "#/properties/countryOfDestination", "type": "Control", "options": {"readOnly": true}}]}, {"scope": "#/properties/totalNetWeight", "type": "Control"}]}',
    '1.0',
    true
);

-- Phytosanitary Certificate
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Phytosanitary Certificate',
    'Phytosanitary certificate for plant products export',
    '{"type": "object", "required": ["distinguishingMarks", "disinfestationTreatment"], "properties": {"countryOfOrigin": {"type": "string", "title": "Country of Origin", "readOnly": true, "x-globalContext": "countryOfOrigin"}, "distinguishingMarks": {"type": "string", "title": "Distinguishing Marks", "example": "BWI-UK-LOT01"}, "countryOfDestination": {"type": "string", "title": "Country of Destination", "readOnly": true, "x-globalContext": "countryOfDestination"}, "disinfestationTreatment": {"type": "string", "title": "Disinfestation Treatment", "example": "Fumigation with Methyl Bromide (CH3Br) at 48g/m³ for 24 hrs"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Phytosanitary Certificate", "type": "Label"}, {"type": "HorizontalLayout", "elements": [{"scope": "#/properties/countryOfOrigin", "type": "Control", "options": {"readOnly": true}}, {"scope": "#/properties/countryOfDestination", "type": "Control", "options": {"readOnly": true}}]}, {"scope": "#/properties/distinguishingMarks", "type": "Control"}, {"scope": "#/properties/disinfestationTreatment", "type": "Control"}]}',
    '1.0',
    true
);

-- Health Certificate
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    '33333333-3333-3333-3333-333333333333',
    'Health Certificate',
    'Health and safety certificate for food products',
    '{"type": "object", "required": ["productDescription", "batchLotNumbers", "productionExpiryDates", "microbiologicalTestReportId", "processingPlantRegistrationNo"], "properties": {"consigneeName": {"type": "string", "title": "Consignee Name", "readOnly": true, "x-globalContext": "consigneeName"}, "batchLotNumbers": {"type": "string", "title": "Batch / Lot Numbers", "description": "DC-2026-JAN-05"}, "consigneeAddress": {"type": "string", "title": "Consignee Address", "readOnly": true, "x-globalContext": "consigneeAddress"}, "productDescription": {"type": "string", "title": "Product Description", "description": "Organic Desiccated Coconut (Fine Grade)"}, "productionExpiryDates": {"type": "string", "title": "Production & Expiry Dates", "description": "Example: Production: YYYY-MM-DD, Expiry: YYYY-MM-DD"}, "microbiologicalTestReportId": {"type": "string", "title": "Microbiological Test Report ID", "description": "ITI/2026/LAB-9982"}, "processingPlantRegistrationNo": {"type": "string", "title": "Processing Plant Registration No.", "description": "CDA/REG/2025/158"}}}',
    '{"type": "VerticalLayout", "elements": [{"text": "Health Certificate", "type": "Label"}, {"scope": "#/properties/consigneeName", "type": "Control", "options": {"readOnly": true}}, {"scope": "#/properties/consigneeAddress", "type": "Control", "options": {"readOnly": true, "multi": true}}, {"scope": "#/properties/productDescription", "type": "Control"}, {"scope": "#/properties/batchLotNumbers", "type": "Control"}, {"scope": "#/properties/productionExpiryDates", "type": "Control"}, {"scope": "#/properties/microbiologicalTestReportId", "type": "Control"}, {"scope": "#/properties/processingPlantRegistrationNo", "type": "Control"}]}',
    '1.0',
    true
);

-- General Information
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES (
    '44444444-4444-4444-4444-444444444444',
    'General Information',
    'General consignment information form',
    '{"type": "object", "title": "General Info", "required": ["consigneeName", "consigneeAddress", "countryOfOrigin", "countryOfDestination"], "properties": {"consigneeName": {"type": "string", "title": "Consignee Name"}, "consigneeAddress": {"type": "string", "title": "Consignee Address"}, "countryOfOrigin": {"enum": ["LK"], "type": "string", "title": "Country of Origin"}, "countryOfDestination": {"type": "string", "title": "Country of Destination"}}}',
    '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/consigneeName", "type": "Control"}, {"scope": "#/properties/consigneeAddress", "type": "Control", "options": {"multi": true}}, {"scope": "#/properties/countryOfOrigin", "type": "Control"}, {"scope": "#/properties/countryOfDestination", "type": "Control"}]}',
    '1.0',
    true
);

-- Placeholder Forms
INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES 
('55555555-5555-5555-5555-555555555555', 'Customs Declaration (Import) - Placeholder', 'Placeholder for import customs declaration', '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}', '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}', '1.0', true),
('66666666-6666-6666-6666-666666666666', 'Delivery Order - Placeholder', 'Placeholder for delivery order', '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}', '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}', '1.0', true),
('77777777-7777-7777-7777-777777777777', 'SLSI Quality Standard Verification - Placeholder', 'Placeholder for SLSI quality verification', '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}', '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}', '1.0', true),
('88888888-8888-8888-8888-888888888888', 'Food Control Unit Health Clearance - Placeholder', 'Placeholder for health clearance', '{"type": "object", "properties": {"placeholder": {"type": "string", "title": "Placeholder"}}}', '{"type": "VerticalLayout", "elements": [{"type": "Label", "text": "Placeholder Form"}]}', '1.0', true);

-- ============================================================================
-- Workflow 1: sl-export-tea-packaged-2.0
-- ============================================================================

INSERT INTO workflow_node_templates (id, type, config, depends_on) VALUES 
('a0000001-0001-0001-0001-000000000001', 'SIMPLE_FORM', '{"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}'::jsonb, '[]'::jsonb),
('a0000001-0001-0001-0001-000000000002', 'SIMPLE_FORM', '{"formId": "22222222-2222-2222-2222-222222222222", "agency": "NPQS", "service": "plant-quarantine"}'::jsonb, '["a0000001-0001-0001-0001-000000000001"]'::jsonb),
('a0000001-0001-0001-0001-000000000003', 'WAIT_FOR_EVENT', '{"agency": "SLTB", "service": "tea-blend-sheet"}'::jsonb, '["a0000001-0001-0001-0001-000000000001"]'::jsonb),
('a0000001-0001-0001-0001-000000000004', 'WAIT_FOR_EVENT', '{"event": "WAIT_FOR_EVENT"}'::jsonb, '["a0000001-0001-0001-0001-000000000002", "a0000001-0001-0001-0001-000000000003"]'::jsonb);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'd299f7e7-eca3-4399-9b22-2ae1d742109d',
    'Sri Lanka Tea Export (Packaged)',
    'Workflow for exporting packaged tea products from Sri Lanka',
    'sl-export-tea-packaged-2.0',
    '["a0000001-0001-0001-0001-000000000001", "a0000001-0001-0001-0001-000000000002", "a0000001-0001-0001-0001-000000000003", "a0000001-0001-0001-0001-000000000004"]'::jsonb
);

INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, workflow_template_id) VALUES 
('b527cff7-9caf-4756-b143-ecf62dbc4236', '6aa146ba-dd72-4e5e-ae27-a1cb5d69caa5', 'EXPORT', 'd299f7e7-eca3-4399-9b22-2ae1d742109d'),
('5a4c798a-d21f-47f6-a257-0c70a556da0e', '2e173ef8-840b-4cc5-a667-03e1d80e04b9', 'EXPORT', 'd299f7e7-eca3-4399-9b22-2ae1d742109d');

-- ============================================================================
-- Workflow 2: sl-import-coconut-oil-2.0
-- ============================================================================

INSERT INTO workflow_node_templates (id, type, config, depends_on) VALUES 
('b0000002-0002-0002-0002-000000000001', 'WAIT_FOR_EVENT', '{"event": "IGM_RECEIVED"}'::jsonb, '[]'::jsonb),
('b0000002-0002-0002-0002-000000000002', 'SIMPLE_FORM', '{"formId": "55555555-5555-5555-5555-555555555555"}'::jsonb, '["b0000002-0002-0002-0002-000000000001"]'::jsonb),
('b0000002-0002-0002-0002-000000000003', 'SIMPLE_FORM', '{"formId": "77777777-7777-7777-7777-777777777777", "agency": "SLSI", "service": "quality-standard-verification"}'::jsonb, '["b0000002-0002-0002-0002-000000000002"]'::jsonb),
('b0000002-0002-0002-0002-000000000004', 'SIMPLE_FORM', '{"formId": "88888888-8888-8888-8888-888888888888", "agency": "MOH", "service": "health-clearance"}'::jsonb, '["b0000002-0002-0002-0002-000000000002"]'::jsonb),
('b0000002-0002-0002-0002-000000000005', 'SIMPLE_FORM', '{"formId": "66666666-6666-6666-6666-666666666666"}'::jsonb, '["b0000002-0002-0002-0002-000000000003", "b0000002-0002-0002-0002-000000000004"]'::jsonb);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    'eea36780-48f2-424c-9b55-0d7394e9677d',
    'Sri Lanka Coconut Oil Import',
    'Workflow for importing coconut oil products to Sri Lanka',
    'sl-import-coconut-oil-2.0',
    '["b0000002-0002-0002-0002-000000000001", "b0000002-0002-0002-0002-000000000002", "b0000002-0002-0002-0002-000000000003", "b0000002-0002-0002-0002-000000000004", "b0000002-0002-0002-0002-000000000005"]'::jsonb
);

INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, workflow_template_id) VALUES 
('74ab852b-5077-4111-abfb-bef40fb5d488', '653c4c8f-8c39-4aee-86f5-7f3926d0d4c2', 'IMPORT', 'eea36780-48f2-424c-9b55-0d7394e9677d'),
('3c2361ca-baf4-4847-82cc-b1cf6d84096f', 'bfa92119-64d3-41f4-b21c-fd0e2eb2966b', 'IMPORT', 'eea36780-48f2-424c-9b55-0d7394e9677d');

-- ============================================================================
-- Workflow 3: sl-export-desiccated-coconut-2.0
-- ============================================================================

INSERT INTO workflow_node_templates (id, type, config, depends_on) VALUES 
('c0000003-0003-0003-0003-000000000001', 'SIMPLE_FORM', '{"formId": "44444444-4444-4444-4444-444444444444"}'::jsonb, '[]'::jsonb),
('c0000003-0003-0003-0003-000000000002', 'SIMPLE_FORM', '{"formId": "11111111-1111-1111-1111-111111111111", "submissionUrl": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec"}'::jsonb, '["c0000003-0003-0003-0003-000000000001"]'::jsonb),
('c0000003-0003-0003-0003-000000000003', 'SIMPLE_FORM', '{"agency": "NPQS", "formId": "22222222-2222-2222-2222-222222222222", "service": "plant-quarantine-phytosanitary", "submissionUrl": "http://localhost:8081/api/oga/inject", "requiresOgaVerification": true}'::jsonb, '["c0000003-0003-0003-0003-000000000002"]'::jsonb),
('c0000003-0003-0003-0003-000000000004', 'SIMPLE_FORM', '{"agency": "EDB", "formId": "33333333-3333-3333-3333-333333333333", "service": "export-product-registration", "submissionUrl": "http://localhost:8082/api/oga/inject", "requiresOgaVerification": true}'::jsonb, '["c0000003-0003-0003-0003-000000000002"]'::jsonb),
('c0000003-0003-0003-0003-000000000005', 'WAIT_FOR_EVENT', '{"event": "WAIT_FOR_EVENT", "externalServiceUrl": "http://localhost:3001/api/process-task"}'::jsonb, '["c0000003-0003-0003-0003-000000000003", "c0000003-0003-0003-0003-000000000004"]'::jsonb);

INSERT INTO workflow_templates (id, name, description, version, nodes) VALUES (
    '44bbe677-d327-4968-bf72-1d314246b486',
    'Sri Lanka Desiccated Coconut Export',
    'Workflow for exporting desiccated coconut products from Sri Lanka',
    'sl-export-desiccated-coconut-2.0',
    '["c0000003-0003-0003-0003-000000000001", "c0000003-0003-0003-0003-000000000002", "c0000003-0003-0003-0003-000000000003", "c0000003-0003-0003-0003-000000000004", "c0000003-0003-0003-0003-000000000005"]'::jsonb
);

INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, workflow_template_id) VALUES 
('5edbd5cb-b347-4272-9541-eee10ce9c387', '36a58d44-8ff6-4bea-8c9b-3db84bb5a083', 'EXPORT', '44bbe677-d327-4968-bf72-1d314246b486'),
('688574a0-e24c-48e4-86eb-1496d5d21da2', '8a0783e4-82e6-488e-b96e-6140a8912f39', 'EXPORT', '44bbe677-d327-4968-bf72-1d314246b486');

-- ============================================================================
-- Migration complete
-- ============================================================================
