-- 1a: Custom Declaration
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000001', 'Customs Declaration (CusDec)', 'Initial submission of export declaration', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000001"}'::jsonb, '[]'::jsonb);

-- 1b: Assessment Notice
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000002', 'Assessment Notice', 'Auto-calculation of Cess and other fees', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000002"}'::jsonb, '["e1111111-1001-4001-a001-000000000001"]'::jsonb);

-- 1c: Payment
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000003', 'Payment on Account', 'Payment confirmation for assessment notice', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000003"}'::jsonb, '["e1111111-1001-4001-a001-000000000002"]'::jsonb);

-- 1d: Warranting
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000004', 'Warranting for Exports', 'Official registration and approval for export processing', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000004"}'::jsonb, '["e1111111-1001-4001-a001-000000000003"]'::jsonb);

-- 2a: Phyto/Health Approval (Parallel to logistics)
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000005', 'Regulatory Approval (PGA)', 'Approval for Phytosanitary or Health Certificates', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000005"}'::jsonb, '["e1111111-1001-4001-a001-000000000001"]'::jsonb);

-- 1e: Selectivity
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000006', 'Risk Selectivity Run', 'Automated risk engine assessment (Green/Red lane)', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000006"}'::jsonb, '["e1111111-1001-4001-a001-000000000004"]'::jsonb);

-- 1f: e-CDN
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000007', 'e-Cargo Dispatch Note', 'Details for Lorry and Truck transport', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000007"}'::jsonb, '["e1111111-1001-4001-a001-000000000004"]'::jsonb);

-- 4: Entry to Yard
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000008', 'Entry to EFC Yard', 'Physical arrival of container at the export yard', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000008"}'::jsonb, '["e1111111-1001-4001-a001-000000000007"]'::jsonb);

-- 5b: Panel Examination
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000009', 'Physical Examination', 'Customs examination result entry', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000009"}'::jsonb, '["e1111111-1001-4001-a001-000000000006", "e1111111-1001-4001-a001-000000000008"]'::jsonb);

-- 6: Export Released (The Convergent Node)
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000010', 'Export Released (Boat Note)', 'Final release to SLPA based on all approvals', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000010"}'::jsonb, '["e1111111-1001-4001-a001-000000000009", "e1111111-1001-4001-a001-000000000005"]'::jsonb);

-- 8: Bill of Lading
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000011', 'Bill of Lading', 'Issuance of shipping document', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000011"}'::jsonb, '["e1111111-1001-4001-a001-000000000010"]'::jsonb);

-- 2b: Final Certificate Issuance
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES ('e1111111-1001-4001-a001-000000000012', 'Certificate Issuance', 'Final issuance of Phyto/Health/COO certificates', 'SIMPLE_FORM', '{"formId": "f0000000-1111-2222-3333-000000000012"}'::jsonb, '["e1111111-1001-4001-a001-000000000011"]'::jsonb);


INSERT INTO workflow_templates (id, name, description, version, nodes)
VALUES (
    'a0000000-9999-8888-7777-666666666666', 
    'NSW National Export Workflow', 
    'Standard operating procedure for Sea Cargo exports via SLPA', 
    '1.0.0', 
    '[
        "e1111111-1001-4001-a001-000000000001", 
        "e1111111-1001-4001-a001-000000000002", 
        "e1111111-1001-4001-a001-000000000003", 
        "e1111111-1001-4001-a001-000000000004", 
        "e1111111-1001-4001-a001-000000000005", 
        "e1111111-1001-4001-a001-000000000006", 
        "e1111111-1001-4001-a001-000000000007", 
        "e1111111-1001-4001-a001-000000000008", 
        "e1111111-1001-4001-a001-000000000009", 
        "e1111111-1001-4001-a001-000000000010", 
        "e1111111-1001-4001-a001-000000000011", 
        "e1111111-1001-4001-a001-000000000012"
    ]'::jsonb
);

INSERT INTO workflow_template_maps (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES ('688574a0-e24c-48e4-86eb-1496d5d21da2', '8a0783e4-82e6-488e-b96e-6140a8912f39', 'EXPORT',
        'a0000000-9999-8888-7777-666666666666');