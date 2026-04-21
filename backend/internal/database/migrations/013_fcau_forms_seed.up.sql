INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES
    -- FCAU APPLICATION FORM
    (
        'fcau-application-form',
        'FCAU Application Form',
        'Form for applicants to submit their application for the FCAU process.',
        '{
            "type": "object",
            "properties": {
                "nameOfApplicant": {
                    "type": "string",
                    "title": "Name of Applicant"
                },
                "addressOfApplicant": {
                    "type": "string",
                    "title": "Address of Applicant"
                },
                "nameOfConsignee": {
                    "type": "string",
                    "title": "Name of Consignee"
                },
                "addressOfConsignee": {
                    "type": "string",
                    "title": "Address of Consignee"
                },
                "descriptionOfConsignment": {
                    "type": "string",
                    "title": "Description of consignment and quantity"
                },
                "dateOfIntendedExport": {
                    "type": "string",
                    "format": "date",
                    "title": "Date of intended export"
                },
                "lcNo": {
                    "type": "string",
                    "title": "L/C No"
                },
                "containerNumbers": {
                    "type": "string",
                    "title": "Upload Container Numbers",
                    "format": "file"
                },
                "nameOfVessel": {
                    "type": "string",
                    "title": "Name of Vessel/ Number"
                },
                "addressConsignmentStored": {
                    "type": "string",
                    "title": "Address where consignment is stored"
                },
                "packageDetails": {
                    "type": "string",
                    "title": "Type of Package, Batch Codes and Total weight"
                },
                "ingredientDetails": {
                    "type": "string",
                    "title": "Upload or Enter Details of ingredients used in product",
                    "format": "file"
                },
                "qualityMonitoringDetails": {
                    "type": "string",
                    "title": "Details of in-house quality monitoring"
                },
                "analysisCertificates": {
                    "type": "string",
                    "title": "Upload Raw materials and product analysis certificates",
                    "format": "file"
                },
                "otherDeclarations": {
                    "type": "string",
                    "title": "Any other Declarations"
                }
            },
            "required": [
                "nameOfApplicant",
                "addressOfApplicant",
                "nameOfConsignee",
                "addressOfConsignee",
                "descriptionOfConsignment",
                "dateOfIntendedExport",
                "lcNo",
                "containerNumbers",
                "nameOfVessel",
                "addressConsignmentStored",
                "packageDetails",
                "ingredientDetails",
                "qualityMonitoringDetails",
                "analysisCertificates"
            ]
        }',
        '{
            "type": "Categorization",
            "elements": [
                {
                    "type": "Category",
                    "label": "Consignment",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/nameOfApplicant"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/addressOfApplicant"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/nameOfConsignee"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/addressOfConsignee"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/dateOfIntendedExport"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/lcNo"
                        },
                        {
                            "type": "VerticalLayout",
                            "elements": [
                                {
                                    "text": "Container Numbers (attach list if necessary)",
                                    "type": "Label"
                                },
                                {
                                    "type": "Control",
                                    "scope": "#/properties/containerNumbers"
                                }
                            ]
                        }
                    ]
                },
                {
                    "type": "Category",
                    "label": "Transport",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/nameOfVessel"
                        }
                    ]
                },
                {
                    "type": "Category",
                    "label": "Items",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/descriptionOfConsignment"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/addressConsignmentStored"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/packageDetails"
                        },
                        {
                            "type": "VerticalLayout",
                            "elements": [
                                {
                                    "text": "Details of ingredients used in product",
                                    "type": "Label"
                                },
                                {
                                    "type": "Control",
                                    "scope": "#/properties/ingredientDetails"
                                }
                            ]
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/qualityMonitoringDetails",
                            "options": {
                                "multi": true
                            }
                        },
                        {
                            "type": "VerticalLayout",
                            "elements": [
                                {
                                    "text": "Raw materials and product analysis certificates (Attach)",
                                    "type": "Label"
                                },
                                {
                                    "type": "Control",
                                    "scope": "#/properties/analysisCertificates"
                                }
                            ]
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/otherDeclarations",
                            "options": {
                                "multi": true
                            }
                        }
                    ]
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU APPLICATION REVIEW RESPONSE FORM
    (
        'fcau-application-review-response',
        'FCAU Application Review Response Form',
        'Form for reviewers to provide their decision and comments on the application.',
        '{
            "type": "object",
            "properties": {
                "applicationId": {
                    "type": "string",
                    "title": "Application ID",
                    "description": "Please enter Application ID"
                }
            },
            "required": [
                "applicationId"
            ]
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/applicationId"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU SAMPLE DROP ACKNOWLEDGEMENT RESPONSE FORM
    (
        'fcau-sample-drop-ack-response',
        'FCAU Sample Drop Acknowledgement Form',
        'Form for lab personnel to acknowledge receipt of samples for testing.',
        '{
            "type": "object",
            "required": [
                "acknowledgement"
            ],
            "properties": {
                "acknowledgement": {
                    "type": "boolean",
                    "title": "Sample Received"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/acknowledgement"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU TESTING REQUIREMENT RESPONSE FORM
    (
        'fcau-testing-requirement-response',
        'FCAU Testing Requirement Response Form',
        'Form for reviewers to provide their decision and comments on the testing requirement.',
        '{
            "type": "object",
            "properties": {
                "labTestingStatus": {
                    "type": "string",
                    "title": "Are lab tests required?",
                    "enum": [
                        "Required",
                        "Not Required"
                    ]
                },
                "requiredTests": {
                    "type": "array",
                    "title": "List Required Lab Tests",
                    "description": "Add a new text box for each specific lab test required.",
                    "items": {
                        "type": "string",
                        "title": "Test Name"
                    }
                }
            },
            "required": [
                "labTestingStatus"
            ]
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/labTestingStatus",
                    "options": {
                        "format": "radio"
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/requiredTests",
                    "rule": {
                        "effect": "SHOW",
                        "condition": {
                            "scope": "#/properties/labTestingStatus",
                            "schema": {
                                "const": "Required"
                            }
                        }
                    }
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU LAB PAYMENT SLIP UPLOAD FORM
    (
        'fcau-lab-payment-form',
        'FCAU Lab Payment Form',
        'Form for applicants to upload proof of payment for lab testing.',
        '{
            "type": "object",
            "properties": {
                "labPaymentReceipt": {
                    "type": "string",
                    "title": "Payment Receipt",
                    "format": "file"
                }
            },
            "required": [
                "labPaymentReceipt"
            ]
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "text": "Upload Lab Payment Receipt",
                    "type": "Label"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/labPaymentReceipt"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU LAB PAYMENT REVIEW RESPONSE FORM
    (
        'fcau-lab-payment-review-response',
        'FCAU Lab Payment Review Response Form',
        'Form for reviewers to verify the lab payment and provide their decision and comments.',
        '{
            "type": "object",
            "required": [
                "paymentReceived"
            ],
            "properties": {
                "paymentReceived": {
                    "type": "boolean",
                    "title": "Payment Received"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/paymentReceived"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU LAB RESULTS REVIEW RESPONSE FORM
    (
        'fcau-lab-results-review-response',
        'FCAU Lab Results Review Response Form',
        'Form for reviewers to review lab results and provide their decision and comments.',
        '{
            "type": "object",
            "required": [
                "decision"
            ],
            "properties": {
                "decision": {
                    "type": "string",
                    "title": "Decision",
                    "oneOf": [
                        {
                            "const": "PASSED",
                            "title": "Pass"
                        },
                        {
                            "const": "FAILED",
                            "title": "Fail"
                        }
                    ]
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/decision"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- FCAU CERTIFICATE VIEW
    (
        'fcau-certificate-issue-response',
        'FCAU Certificate Issue Response Form',
        'Form for reviewers to provide their decision and comments on the certificate issue.',
        '{
            "type": "object",
            "required": [
                "certificate"
            ],
            "properties": {
                "certificate": {
                    "type": "string",
                    "title": "Upload Certificate",
                    "format": "file"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/certificate"
                }
            ]
        }',
        '1.0',
        TRUE
    );