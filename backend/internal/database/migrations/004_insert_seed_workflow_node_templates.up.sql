-- ============================================================================
-- Migration: 001_insert_seed_workflow_node_templates.sql
-- Purpose: Seed workflow node templates and unlock rules for execution flow.
-- ============================================================================

-- Seed data: workflow node template catalog
INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on, unlock_configuration)
VALUES
    -- General Information Node
    (
        'c0000003-0003-0003-0003-000000000001',
        'General Information',
        'General consignment information form',
        'SIMPLE_FORM',
        '{
            "formId": "44444444-4444-4444-4444-444444444444"
        }',
        '[]',
        NULL
    ),

    -- Customs Declaration Node
    (
        'c0000003-0003-0003-0003-000000000002',
        'Customs Declaration',
        'Export customs declaration form for trade goods',
        'SIMPLE_FORM',
        '{
            "formId": "11111111-1111-1111-1111-111111111111",
            "submission": {
                "url": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec",
                "response": {
                    "display": {
                        "formId": "71e5e73a-ebb2-4750-aaa4-f71087adac43"
                    },
                    "mapping": {
                        "assesmentNo": "cusdec:assesmentNo"
                    }
                }
            }
        }',
        '[
            "c0000003-0003-0003-0003-000000000001"
        ]',
        '{
            "expression": {
                "state": "COMPLETED",
                "nodeTemplateId": "c0000003-0003-0003-0003-000000000001"
            }
        }'
    ),

    -- Payment Node
    (
        'c0000003-0003-0003-0003-000000000008',
        'Payment',
        'Base payment step before customs declaration',
        'PAYMENT',
        '{
                  "currency": "LKR",
                  "ttl": 3600,
                  "orgId": "CUSTOMS",
                  "serviceType": "CUSTOMS DECLARATION",
                  "breakdown": [
                    {
                      "description": "Levy Payment for {cusdec.id}",
                      "category": "ADDITION",
                      "type": "FIXED",
                      "quantity": "{gx_quantity_levy:1}",
                      "unitPrice": "{cusdec.cess:345}"
                    },
                    {
                      "description": "Processing Fee",
                      "category": "ADDITION",
                      "type": "FIXED",
                      "quantity": "1",
                      "unitPrice": "500.00"
                    },
                    {
                      "description": "Exemption",
                      "category": "DEDUCTION",
                      "type": "PERCENTAGE",
                      "value": "5"
                    },
                    {
                      "description": "VAT",
                      "category": "ADDITION",
                      "type": "PERCENTAGE",
                      "value": "{vat_rate:15}"
                    }
                  ]
                }',
        '[
            "c0000003-0003-0003-0003-000000000002"
        ]',
        '{
            "expression": {
                "state": "COMPLETED",
                "nodeTemplateId": "c0000003-0003-0003-0003-000000000002"
            }
        }'
    ),

    -- Phytosanitary Certificate Node
    (
        'c0000003-0003-0003-0003-000000000003',
        'Phytosanitary Certificate',
        'Phytosanitary certificate for plant products export',
        'SIMPLE_FORM',
        ('{
            "agency": "NPQS",
            "formId": "22222222-2222-2222-2222-222222222222",
            "service": "plant-quarantine-phytosanitary",
            "callback": {
                "response": {
                    "display": {
                        "formId": "d0c3b860-635b-4124-8081-d3f421e429cb"
                    },
                    "mapping": {
                        "reviewedAt": "gi:phytosanitary:meta:reviewedAt",
                        "reviewerNotes": "gi:phytosanitary:meta:reviewNotes"
                    }
                },
                "transition": {
                    "field": "decision",
                    "default": "OGA_VERIFICATION_REJECTED",
                    "mapping": {
                        "APPROVED": "OGA_VERIFICATION_APPROVED",
                        "MANUAL_REVIEW": "OGA_VERIFICATION_APPROVED"
                    }
                }
            },
            "emission": {
                "rules": [
                    {
                        "outcome": "npqs:phytosanitary:high_risk_manual_review",
                        "conditions": [
                            {
                                "field": "ogaResponse.decision",
                                "value": "MANUAL_REVIEW"
                            },
                            {
                                "field": "submissionResponse.riskLevel",
                                "value": "HIGH"
                            }
                        ]
                    },
                    {
                        "outcome": "npqs:phytosanitary:manual_review_required",
                        "conditions": [
                            {
                                "field": "ogaResponse.decision",
                                "value": "MANUAL_REVIEW"
                            }
                        ]
                    },
                    {
                        "outcome": "npqs:phytosanitary:approved",
                        "conditions": [
                            {
                                "field": "ogaResponse.decision",
                                "value": "APPROVED"
                            }
                        ]
                    }
                ]
            },
            "submission": {
                "url": ' || to_jsonb((:'NPQS_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "npqs:phytosanitary:v1"
                }
            }
        }')::jsonb,
        '[
            "c0000003-0003-0003-0003-000000000008"
        ]',
        '{
            "expression": {
                "state": "COMPLETED",
                "nodeTemplateId": "c0000003-0003-0003-0003-000000000008"
            }
        }'
    ),

    -- Health Certificate Node
    (
        'c0000003-0003-0003-0003-000000000004',
        'Health Certificate',
        'Health and safety certificate for food products',
        'SIMPLE_FORM',
        ('{
            "agency": "EDB",
            "formId": "33333333-3333-3333-3333-333333333333",
            "service": "food-control-administration-unit",
            "callback": {
                "response": {
                    "display": {
                        "formId": "95d7e7fe-5be0-43cb-ac71-94bc70d3a01d"
                    }
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "fcau:health_certificate:v1"
                }
            }
        }')::jsonb,
        '[
            "c0000003-0003-0003-0003-000000000008"
        ]',
        '{
            "expression": {
                "state": "COMPLETED",
                "nodeTemplateId": "c0000003-0003-0003-0003-000000000008"
            }
        }'
    ),

    -- Manual Inspection Node (conditional based on phytosanitary certificate outcome)
    (
        'e1a00001-0001-4000-b000-000000000007',
        'Manual Inspection',
        'Manual inspection task for high-risk phytosanitary cases',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Awaiting Physical Inspection",
                "description": "Consignment flagged for physical inspection. NPQS inspector will review the consignment on-site."
            },
            "submission": {
                "url": ' || to_jsonb((:'NPQS_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "npqs:manual_inspection:v1",
                    "template": {
                        "consignee_name": "consignee:consignee_name",
                        "consigneeAddress": "consignee:address"
                    }
                },
                "response": {
                    "display": {
                        "formId": "f1a00001-0001-4000-c000-000000000002"
                    },
                    "mapping": {
                        "inspectionDecision": "npqs:manual_inspection:decision",
                        "inspectorRemarks": "npqs:manual_inspection:remarks"
                    }
                }
            }
        }')::jsonb,
        '[
            "c0000003-0003-0003-0003-000000000003"
        ]',
        '{
            "expression": {
                "allOf": [
                    {
                        "state": "COMPLETED",
                        "nodeTemplateId": "c0000003-0003-0003-0003-000000000003"
                    },
                    {
                        "outcome": "npqs:phytosanitary:manual_review_required",
                        "nodeTemplateId": "c0000003-0003-0003-0003-000000000003"
                    }
                ]
            }
        }'
    ),

    -- Final Processing Node (unlocks when both certificates are completed, or if customs was fast-tracked)
    (
        'e1a00001-0001-4000-b000-000000000005',
        'Final Processing',
        'Final processing step — unlocks when both certificates are completed, or customs was fast-tracked',
        'WAIT_FOR_EVENT',
        '{
            "display": {
                "title": "Waiting for ship to leave from port",
                "description": "The task will be completed when the ship leaves the port. This is an external event that we are waiting for."
            },
            "submission": {
                "url": "http://localhost:8081/api/oga/inject",
                "request": {
                    "taskCode": "ship_departure_v1",
                    "template": {
                        "port_code": "departure_port",
                        "vessel_id": "ship_identifier"
                    }
                },
                "response": {
                    "display": {
                        "formId": "95d7e7fe-5be0-43cb-ac71-94bc70d3a01d"
                    }
                }
            }
        }',
        '[
            "c0000003-0003-0003-0003-000000000003",
            "c0000003-0003-0003-0003-000000000004",
            "e1a00001-0001-4000-b000-000000000007"
        ]',
        '{
            "expression": {
                "allOf": [
                    {
                        "anyOf": [
                            {
                                "allOf": [
                                    {
                                        "state": "COMPLETED",
                                        "nodeTemplateId": "c0000003-0003-0003-0003-000000000003"
                                    },
                                    {
                                        "outcome": "npqs:phytosanitary:approved",
                                        "nodeTemplateId": "c0000003-0003-0003-0003-000000000003"
                                    }
                                ]
                            },
                            {
                                "state": "COMPLETED",
                                "nodeTemplateId": "e1a00001-0001-4000-b000-000000000007"
                            }
                        ]
                    },
                    {
                        "state": "COMPLETED",
                        "nodeTemplateId": "c0000003-0003-0003-0003-000000000004"
                    }
                ]
            }
        }'
    ),

    -- Placeholder End Node (to satisfy end_node_template_id requirement)
    (
        'e1a00001-0001-4000-b000-000000000006',
        'End Node',
        'Placeholder end node template to satisfy end_node_template_id requirement',
        'END_NODE',
        '{}',
        '[]',
        NULL
    ) ON CONFLICT (id) DO NOTHING;

