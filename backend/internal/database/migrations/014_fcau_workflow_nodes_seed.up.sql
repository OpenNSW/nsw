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
                    "taskCode": "fcau_general_application_v1"
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
                "title": { 
                    "waiting": "Waiting on Sample Drop Off Confirmation",
                    "failed": "Sample Drop Off Confirmation Failed",
                    "completed": "Sample Drop Off Confirmation Completed"
                },
                "description": {
                    "waiting": "Please drop off your sample at the designated location and confirm the drop off by clicking on this task",
                    "failed": "We have not received confirmation of your sample drop off. Please confirm that you have dropped off your sample by clicking on this task.",
                    "completed": "We have received confirmation of your sample drop off. We will notify you once the test results are available."
                 }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "fcau_sample_drop_ack_v1",
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
                "title": {
                    "waiting": "Waiting on Testing Requirements",
                    "failed": "Testing Requirement Analysis Failed",
                    "completed": "Testing Requirement Analysis Completed"
                },
                "description": {
                    "waiting": "Once the FCAU officer decides on the testing requirements, this task will get completed",
                    "failed": "There was an issue determining the testing requirements. Please retry.",
                    "completed": "The FCAU officer has determined the testing requirements based on your submission. If lab testing is required, you will see a new task to upload payment for the lab testing."
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "fcau_testing_requirement_v1",
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
                    "taskCode": "fcau_lab_payment_upload_v1"
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
                "title": {
                    "waiting": "Waiting on Test Result Evaluation",
                    "failed": "Test Result Evaluation Failed",
                    "completed": "Test Result Evaluation Completed"
                },
                "description": {
                    "waiting": "Once the FCAU officer reviews the lab test results, this task will be marked as complete",
                    "failed": "There was an issue evaluating the test results. Please retry.",
                    "completed": "The FCAU officer has reviewed the test results and made a decision. You can view the decision details by clicking on this task."
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "fcau_lab_results_review_v1",
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
                "title": {
                    "waiting": "Waiting on Certificate Issuing",
                    "failed": "Certificate Issuance Failed",
                    "completed": "Certificate Issued"
                },
                "description": {
                    "waiting": "Once the FCAU officer issues the certificate, you will be able to view it here",
                    "failed": "There was an issue issuing the certificate. Please retry.",
                    "completed": "Your certificate has been issued successfully. You can view it in the attachments section below."
                }            
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "taskCode": "fcau_certificate_issue_v1",
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

