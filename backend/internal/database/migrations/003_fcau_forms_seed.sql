INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES
    (
        'fcau-application-form',
        'FCAU Application Form',
        'Form for applicants to submit their application for the FCAU process.',
        '{
            "type": "object",
            "properties": {},
            "required": []
        }'::jsonb,
        '{}'::jsonb,
        1,
        true
    ),
    (
        'fcau-lab-payment-form',
        'FCAU Lab Payment Form',
        'Form for applicants to upload proof of payment for lab testing.',
        '{
            "type": "object",
            "properties": {},
            "required": []
        }'::jsonb,
        '{}'::jsonb,
        1,
        true
    ),
    (
        'fcau-application-review-response',
        'FCAU Application Review Response Form',
        'Form for reviewers to provide their decision and comments on the application.',
        '{
            "type": "object",
            "properties": {
                "decision": {
                    "type": "string",
                    "enum": ["APPROVED", "REJECTED"]
                },
                "reviewer_comments": {
                    "type": "string"
                }
            },
            "required": ["decision"]
        }'::jsonb,
        '{}'::jsonb,
        1,
        true
    ),
    (
        'fcau-lab-payment-review-response',
        'FCAU Lab Payment Review Response Form',
        'Form for reviewers to provide their decision and comments on the lab payment.',
        '{
            "type": "object",
            "properties": {
                "decision": {
                    "type": "string",
                    "enum": ["APPROVED", "REJECTED"]
                },
                "reviewer_comments": {
                    "type": "string"
                }
            },
            "required": ["decision"]
         }'::jsonb,
        '{}'::jsonb,
        1,
        true
    );