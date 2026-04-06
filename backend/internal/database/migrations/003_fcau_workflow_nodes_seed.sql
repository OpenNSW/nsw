INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES
    -- FCAU Application Submission Task
    (
        'fcau:application_submission',
        'Application Submission',
        'Task for applicants to submit their application for the FCAU process.',
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
                        "decision": "fcau:application_decision",
                        "reviewer_comments": "fcau:reviewer_comments"
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
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
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
        'Sample Drop-off',
        'Task for applicants to confirm they have dropped off their sample at the designated location.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- FCAU Testing Requirement Task
    (
        'fcau:testing_requirement',
        'Testing Requirement Assessment',
        'Task to determine if the applicant requires lab testing based on their application details.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
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
                    "default": "OGA_VERIFICATION_REJECTED",
                    "mapping": {
                        "APPROVED": "OGA_VERIFICATION_APPROVED",
                        "MANUAL_REVIEW": "OGA_VERIFICATION_APPROVED"
                    }
                }
            },
            "submission": {
                "url": ' || to_jsonb((:'FCAU_OGA_SUBMISSION_URL')::text)::text || ',
                "request": {
                    "meta": {
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
        '{}'::jsonb,
        '[]'
    ),

    -- FCAU Payment Task
    (
        'fcau:payment',
        'Payment',
        'Task for applicants to make payment for the FCAU process.',
        'PAYMENT',
        '{}'::jsonb,
        '[]'
    ),

    -- FCAU Certificate Issue Task
    (
        'fcau:certificate_issue',
        'Certificate Issuance',
        'Task for issuing the certificate to the applicant upon successful completion of the process.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    );

