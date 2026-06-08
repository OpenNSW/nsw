INSERT INTO workflow_template_v2 (id, name, version, workflow_definition)
VALUES
    (
        'fcau-health-certificate-reg',
        'FCAU Export Consignment & Health Certificate Registration',
        '1',
        '{}'::jsonb
    );

-- Seed test HS codes starting with 'fc' for local testing
INSERT INTO hs_codes (id, hs_code, description, category)
VALUES
    (
        'fcau-hs-code-0001',
        'fcau-health-cert',
        'HS code representing the FCAU process for testing.',
        'FCAU'
    );

INSERT INTO workflow_template_map (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES
    -- Mapping for FCAU process
    (
        'fcau-wf-map-0001',
        'fcau-hs-code-0001',
        'EXPORT',
        'fcau-health-certificate-reg'
    );