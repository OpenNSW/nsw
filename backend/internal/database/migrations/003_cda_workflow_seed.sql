INSERT INTO workflow_template_v2 (id, name, version, workflow_definition)
VALUES
(
    'cda-v1',
    'Issuance of Salmonella Free and Physical Quality Certificates',
    '1',
    '{
        "id": "cda-v1",
        "name": "Issuance of Salmonella Free and Physical Quality Certificates",
        "version": 1,
        "nodes": [
            {
                "id": "node_start",
                "name": "Start",
                "type": "START"
            },
            {
                "id": "node_submit_app",
                "name": "[CDAPI 2.0] Submit Application",
                "type": "TASK",
                "task_template_id": "cda:application_submission",
                "output_mapping": {
                    "application_id": "cda.application_id"
                }
            },
            {
                "id": "node_make_payment",
                "name": "[CDAPI 3.0] Make Payment",
                "type": "TASK",
                "task_template_id": "cda:payment",
                "input_mapping": {
                    "cda.application_id": "application_id"
                },
                "output_mapping": {
                    "payment_reference": "cda.payment_reference"
                }
            },
            {
                "id": "node_wait_cert",
                "name": "[CDAPI 4.0] Wait on Certificate Issuing",
                "type": "TASK",
                "task_template_id": "cda:certificate_issue",
                "input_mapping": {
                    "cda.application_id": "application_id"
                },
                "output_mapping": {
                    "salmonella_cert_uri": "cda.salmonella_cert_uri",
                    "quality_cert_uri": "cda.quality_cert_uri"
                }
            },
            {
                "id": "node_end",
                "name": "End",
                "type": "END"
            }
            ],
            "edges": [
            {
                "id": "edge_1",
                "source_id": "node_start",
                "target_id": "node_submit_app"
            },
            {
                "id": "edge_2",
                "source_id": "node_submit_app",
                "target_id": "node_make_payment"
            },
            {
                "id": "edge_3",
                "source_id": "node_make_payment",
                "target_id": "node_wait_cert"
            },
            {
                "id": "edge_4",
                "source_id": "node_wait_cert",
                "target_id": "node_end"
            }
        ]
    }'::jsonb
);


-- Purpose: Seed workflow templates and mappings for the CDA process.
INSERT INTO hs_codes (id, hs_code, description, category)
VALUES
    (
        'cda-hs-code-0001',
        'cda-process',
        'HS code representing the CDA process for testing.',
        'CDA'
    );

INSERT INTO workflow_template_maps_v2 (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES
    -- Mapping for CDA process
    (
        'cda-wf-map-0001',
        'cda-hs-code-0001',
        'EXPORT',
        'cda-v1'
    );