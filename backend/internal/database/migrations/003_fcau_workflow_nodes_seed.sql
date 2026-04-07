INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES
    -- FCAU Application Submission Task
    (
        'fcau:application_submission',
        'Application Submission',
        'Task for applicants to submit their application for the FCAU process',
        'SIMPLE_FORM',
        ('{
            "agency": "FCAU",
            "formId": "fcau-application-form",
            "service": "food-control-administration-unit",
            "callback": {
                "response": {
                    "display": {
                        "formId": "fcau-application-review-response"
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
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "SIMPLE_FORM",
                        "templateKey": "fcau:general_application:v1"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Sample Drop Task
    (
        'fcau:sample_drop',
        'Sample Drop Off Confirmation',
        'Task to confirm with the applicant if they have dropped off their sample for testing',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Drop off sample",
                "description": "Please drop off your sample at the designated location"
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "fcau:sample_drop_ack:v1"
                    },
                    "template": {
                        "Application ID": "application_id"
                    }
                },
                "response": {
                    "display": {
                        "formId": "fcau-sample-drop-ack-response"
                    },
                    "mapping": {
                        "acknowledgement": "sample_drop_confirmed"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Testing Requirement Task
    (
        'fcau:testing_requirement',
        'Analyze for Testing Requirement',
        'Task to determine if the applicant requires lab testing based on the submitted information',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Waiting on Testing Requirements",
                "description": "Once the FCAU officer decides on the testing requirements, this task will get completed"
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "fcau:testing_requirement:v1"
                    },
                    "template": {
                        "Application ID": "application_id"
                    }
                },
                "response": {
                    "display": {
                        "formId": "fcau-testing-requirement-response"
                    },
                    "mapping": {
                        "labTestingStatus": "lab_testing_status",
                        "requiredTests": "required_tests"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Lab Payment Upload Task
    (
        'fcau:lab_payment_upload',
        'Lab Payment Upload',
        'Task for applicants to upload proof of payment for lab testing.',
        'SIMPLE_FORM',
        ('{
            "agency": "FCAU",
            "formId": "fcau-lab-payment-form",
            "service": "food-control-administration-unit",
            "callback": {
                "response": {
                    "display": {
                        "formId": "fcau-lab-payment-review-response"
                    },
                    "mapping": {
                        "decision": "fcau:lab_payment_decision",
                        "reviewer_comments": "fcau:reviewer_comments"
                    }
                },
                "transition": {
                    "field": "decision",
                    "default": "OGA_VERIFICATION_APPROVED",
                    "mapping": {
                        "false" : "OGA_VERIFICATION_REJECTED"
                    }
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "SIMPLE_FORM",
                        "templateKey": "fcau:lab_payment_upload:v1"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Lab Results Review Task
    (
        'fcau:lab_results_review',
        'Lab Results Review',
        'Task for lab personnel to review the test results and make a decision.',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": " Waiting on Test Result Evaluation",
                "description": "Once the FCAU officer reviews the lab test results, this task will be marked as complete"
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "fcau:lab_results_review:v1"
                    },
                    "template": {
                        "Application ID": "application_id",
                        "Required Tests": "required_tests"
                    }
                },
                "response": {
                    "display": {
                        "formId": "lab-results-review-response"
                    },
                    "mapping": {
                        "decision": "lab_decision"
                    }
                }
            }
        }')::jsonb,
        '[]'
    ),

    -- FCAU Payment Task
    (
        'fcau:payment',
        'Payment',
        'Task for applicants to make payment for the FCAU process.',
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

    -- FCAU Certificate Issue Task
    (
        'fcau:certificate_issue',
        'Certificate Issuance',
        'Task for issuing the certificate to the applicant upon successful completion of the process',
        'WAIT_FOR_EVENT',
        ('{
            "display": {
                "title": "Waiting on Certificate Issuing",
                "description": "Once the FCAU officer issues the certificate, you will be able to view it here"
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
                        "type": "WAIT_FOR_EVENT",
                        "templateKey": "fcau:certificate_issue:v1"
                    },
                    "template": {
                        "Application ID": "application_id"
                    }
                },
                "response": {
                    "display": {
                        "formId": "fcau-certificate-issue-response"
                    },
                    "mapping": {
                        "certificate": "fcau:certificate"
                    }
                }
            }
        }')::jsonb,
        '[]'
    );

