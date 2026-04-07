INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES
    -- CDS Application Submission Task
    (
        'cda:application_submission',
        'Application Submission',
        'Task for applicants to submit their application for the CDA process',
        'SIMPLE_FORM',
        ('{
            "agency": "CDA",
            "formId": "cda-application-form",
            "service": "food-control-administration-unit",
            "callback": {
                "response": {
                    "display": {
                        "formId": "cda-application-review-response"
                    },
                    "mapping": {
                        "applicationId": "application_id"
                    }
                },
                "transition": {
                    "field": "decision",
                    "default": "OGA_VERIFICATION_APPROVED",
                    "mapping": {
                        "REJECTED": "OGA_VERIFICATION_REJECTED"
                    }
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'CDA_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "SIMPLE_FORM",
                        "templateKey": "cda:general_application:v1"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- CDA Payment Task
    (
        'cda:payment',
        'Payment',
        'Task for applicants to make payment for the CDA process.',
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
        '[]'
    ),

    -- CDA Certificate Issue Task
    (
        'cda:certificate_issue',
        'Certificate Issuance',
        'Task for issuing the certificate to the applicant upon successful completion of the process',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Waiting on Certificate Issuing",
                "description": "Once the CDA officer issues the certificate, you will be able to view it here"
            },
            "submission": {
                "url": ' || to_jsonb((:'CDA_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "cda:certificate_issue:v1"
                    },
                    "template": {
                        "Application ID": "application_id"
                    }
                },
                "response": {
                    "display": {
                        "formId": "cda-certificate-issue-response"
                    },
                    "mapping": {
                        "certificate": "cda:certificate"
                    }
                }
            }
        }')::jsonb,
        '[]'
    );

