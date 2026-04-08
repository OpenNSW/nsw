INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES
    -- CDA APPLICATION FORM
    (
        'cda-application-form',
        'CDA Application Form',
        'Form for applicants to submit their application for the CDA process.',
        '{
            "type": "object",
            "properties": {
                "paymentDetails": {
                    "type": "string",
                    "title": "Upload Payment Details",
                    "format": "file"
                },
                "paymentCategory": {
                    "type": "string",
                    "enum": [
                        "General",
                        "SVAT",
                        "Value Added DC/ LAW FAT"
                    ],
                    "description": "Payment Category"
                },
                "exporterNameAddressReg": {
                    "type": "string",
                    "enum": [
                        "Exporter A",
                        "Exporter B"
                    ],
                    "description": "Exporter Name, Address & Registration Number",
                    "x-globalContext": {
                        "writeTo": "exporter_name_address_reg"
                    }
                },
                "paymentSlip": {
                    "type": "string",
                    "title": "Upload Payment Slip",
                    "format": "file"
                },
                "contactNumber": {
                    "type": "number",
                    "description": "Contact Number"
                },
                "cusdecNumber": {
                    "type": "string",
                    "description": "CUSDEC Number"
                },
                "hsCode": {
                    "type": "number",
                    "description": "HS CODE"
                },
                "contractNumber": {
                    "type": "number",
                    "description": "Contract Number"
                },
                "lotNumber": {
                    "type": "number",
                    "description": "Lot Number"
                },
                "dcGrade": {
                    "title": "DC Grade",
                    "description": "DC Grade",
                    "type": "array",
                    "items": {
                        "type": "string",
                        "enum": [
                            "FINE",
                            "MEDIUM",
                            "COARSE"
                        ]
                    },
                    "uniqueItems": true
                },
                "numberOfBags": {
                    "type": "number",
                    "description": "Number of Bags"
                },
                "weightOfBag": {
                    "type": "number",
                    "description": "Weight of a Bag"
                },
                "totalWeight": {
                    "type": "number",
                    "description": "Total Weight"
                },
                "dateOfContainerization": {
                    "type": "string",
                    "format": "date",
                    "description": "Date of Containerization"
                },
                "nameOfVessel": {
                    "type": "string",
                    "description": "Name of the Vessel"
                },
                "containerNumber": {
                    "type": "string",
                    "description": "Container Number"
                },
                "dateOfSailing": {
                    "type": "string",
                    "format": "date",
                    "description": "Date of Sailing"
                },
                "portOfDischarge": {
                    "type": "string",
                    "description": "Port of Discharge"
                },
                "finalDestination": {
                    "type": "string",
                    "description": "Final Destination"
                },
                "nameOfBuyer": {
                    "type": "string",
                    "description": "Name of the Buyer"
                },
                "containerFillingStatus": {
                    "type": "string",
                    "enum": [
                        "Full",
                        "Part"
                    ],
                    "description": "Container Filling Status"
                },
                "balanceMaterialNature": {
                    "type": "string",
                    "description": "If Part, Declare Nature of Balance Material"
                },
                "requiredCertificates": {
                    "type": "array",
                    "title": "pick all certificates that are being requested",
                    "x-globalContext": {
                        "writeTo": "required_certificates"
                    },
                    "description": "pick all certificates that are being requested",
                    "items": {
                        "type": "string",
                        "enum": [
                            "SALMONELLA",
                            "OTHER"
                        ]
                    },
                    "uniqueItems": true

                },
                "detailsOfBags": {
                    "type": "string",
                    "title": "Upload Details of Bags (Please Fill Google Sheet sent with this Form)",
                    "format": "file"
                }
            },
            "required": [
                "paymentDetails",
                "paymentCategory",
                "exporterNameAddressReg",
                "paymentSlip",
                "contactNumber",
                "cusdecNumber",
                "dcGrade",
                "numberOfBags",
                "weightOfBag",
                "totalWeight",
                "portOfDischarge",
                "finalDestination",
                "containerFillingStatus",
                "detailsOfBags"
            ]
        }',
        '{
            "type": "Categorization",
            "elements": [
                {
                    "type": "Category",
                    "label": "General",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/contactNumber"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/contractNumber"
                        }
                    ]
                },
                {
                    "type": "Category",
                    "label": "Consignment",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/exporterNameAddressReg"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/cusdecNumber"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/hsCode"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/lotNumber"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/dcGrade"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/numberOfBags"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/weightOfBag"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/totalWeight"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/dateOfContainerization"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/nameOfVessel"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/containerNumber"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/dateOfSailing"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/portOfDischarge"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/finalDestination"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/nameOfBuyer"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/containerFillingStatus",
                            "options": {
                                "format": "radio"
                            }
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/balanceMaterialNature",
                            "options": {
                                "multi": true
                            }
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/requiredCertificates",
                            "label": "pick all certificates that are being requested"
                        },
                        {
                            "type": "VerticalLayout",
                            "elements": [
                                {
                                    "text": "Uploaded Details of Bags",
                                    "type": "Label"
                                },
                                {
                                    "type": "Control",
                                    "scope": "#/properties/detailsOfBags"
                                }
                            ]
                        }
                    ]
                },
                {
                    "type": "Category",
                    "label": "Finance",
                    "elements": [
                        {
                            "type": "VerticalLayout",
                            "elements": [
                                {
                                    "text": "Uploaded Payment Details",
                                    "type": "Label"
                                },
                                {
                                    "type": "Control",
                                    "scope": "#/properties/paymentDetails"
                                }
                            ]
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/paymentCategory",
                            "options": {
                                "format": "radio"
                            }
                        },
                        {
                            "type": "VerticalLayout",
                            "elements": [
                                {
                                    "text": "Uploaded Payment Slip",
                                    "type": "Label"
                                },
                                {
                                    "type": "Control",
                                    "scope": "#/properties/paymentSlip"
                                }
                            ]
                        }
                    ]
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- CDA APPLICATION REVIEW RESPONSE FORM
    (
        'cda-application-review-response',
        'CDA Application Review Response Form',
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

    -- CDA CERTIFICATE VIEW
    (
        'cda-certificate-issue-response',
        'CDA Certificate Issue Response Form',
        'Form for reviewers to provide their decision and comments on the certificate issue.',
        '{
            "type": "object",
            "properties": {
                "certificates": {
                    "type": "array",
                    "title": "Uploaded Certificates",
                    "description": "Add and upload all necessary certificates below.",
                    "minItems": 1,
                    "items": {
                        "type": "object",
                        "properties": {
                            "certificateName": {
                                "type": "string",
                                "title": "Certificate Name / Description"
                            },
                            "certificateFile": {
                                "type": "string",
                                "title": "Upload Document",
                                "format": "file"
                            }
                        },
                        "required": [
                            "certificateFile"
                        ]
                    }
                }
            },
            "required": [
                "certificates"
            ]
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Label",
                    "text": "Please upload all requested certificates"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/certificates"
                }
            ]
        }',
        '1.0',
        TRUE
    );