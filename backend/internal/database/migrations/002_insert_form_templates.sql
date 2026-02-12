-- 1a: Customs Declaration Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000001', 'Customs Declaration', 'Initial declaration submission', 
        '{"type": "object", "properties": {"cusdecNumber": {"type": "string", "title": "CusDec Number"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/cusdecNumber", "type": "Control"}]}', '1.0', true);

-- 1b: Assessment Notice Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000002', 'Assessment Notice', 'Fee calculation details', 
        '{"type": "object", "properties": {"totalPayable": {"type": "number", "title": "Total Fees Payable (LKR)"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/totalPayable", "type": "Control"}]}', '1.0', true);

-- 1c: Payment Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000003', 'Payment Receipt', 'Payment verification', 
        '{"type": "object", "properties": {"receiptNumber": {"type": "string", "title": "Bank Reference Number"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/receiptNumber", "type": "Control"}]}', '1.0', true);

-- 1d: Warranting Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000004', 'Warranting Status', 'Official warranting registration', 
        '{"type": "object", "properties": {"warrantingDate": {"type": "string", "format": "date", "title": "Warranting Date"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/warrantingDate", "type": "Control"}]}', '1.0', true);

-- 2a: Regulatory Approval Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000005', 'PGA Approval', 'Partner Government Agency approval status', 
        '{"type": "object", "properties": {"pgaStatus": {"type": "string", "enum": ["APPROVED", "REJECTED"], "title": "Regulatory Status"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/pgaStatus", "type": "Control"}]}', '1.0', true);

-- 1e: Risk Selectivity Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000006', 'Risk Assessment', 'Selectivity engine output', 
        '{"type": "object", "properties": {"riskLane": {"type": "string", "enum": ["GREEN", "YELLOW", "RED"], "title": "Assigned Risk Lane"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/riskLane", "type": "Control"}]}', '1.0', true);

-- 1f: e-CDN Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000007', 'e-Cargo Dispatch', 'Transport and vehicle details', 
        '{"type": "object", "properties": {"vehicleNumber": {"type": "string", "title": "Truck/Lorry Number"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/vehicleNumber", "type": "Control"}]}', '1.0', true);

-- 4: Entry to Yard Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000008', 'Gate Entry', 'Confirmation of arrival at terminal', 
        '{"type": "object", "properties": {"gateWeight": {"type": "number", "title": "Weighbridge Weight (kg)"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/gateWeight", "type": "Control"}]}', '1.0', true);

-- 5b: Physical Examination Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000009', 'Customs Examination', 'Results of physical inspection', 
        '{"type": "object", "properties": {"examResult": {"type": "string", "title": "Examination Remarks"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/examResult", "type": "Control"}]}', '1.0', true);

-- 6: Export Released Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000010', 'Boat Note / ERO', 'Final release authorization', 
        '{"type": "object", "properties": {"releaseOrderNumber": {"type": "string", "title": "Electronic Release Number"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/releaseOrderNumber", "type": "Control"}]}', '1.0', true);

-- 8: Bill of Lading Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000011', 'Bill of Lading Details', 'Shipping document reference', 
        '{"type": "object", "properties": {"blNumber": {"type": "string", "title": "Bill of Lading Number"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/blNumber", "type": "Control"}]}', '1.0', true);

-- 2b: Final Certificate Issuance Form
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES ('f0000000-1111-2222-3333-000000000012', 'Certificate Issuance', 'Final issuance of regulatory papers', 
        '{"type": "object", "properties": {"certificateUrl": {"type": "string", "title": "Download Link for Certificate"}}}', 
        '{"type": "VerticalLayout", "elements": [{"scope": "#/properties/certificateUrl", "type": "Control"}]}', '1.0', true);