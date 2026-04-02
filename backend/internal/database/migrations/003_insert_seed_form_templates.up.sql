-- ============================================================================
-- Migration: 001_insert_seed_form_templates.sql
-- Purpose: Seed baseline form templates and supporting response/review forms.
-- ============================================================================

-- Seed data: form template catalog
INSERT INTO forms (id, name, description, schema, ui_schema, version, active)
VALUES
    -- Customs Declaration
    (
        '11111111-1111-1111-1111-111111111111',
        'Customs Declaration',
        'Export customs declaration form for trade goods',
        '{
            "type": "object",
            "required": [
                "declarationType",
                "totalInvoiceValue",
                "totalPackages",
                "totalNetWeight"
            ],
            "properties": {
                "totalPackages": {
                    "type": "number",
                    "title": "Total Packages",
                    "minimum": 0
                },
                "totalNetWeight": {
                    "type": "number",
                    "title": "Total Net Weight (kg)",
                    "minimum": 0
                },
                "countryOfOrigin": {
                    "type": "string",
                    "title": "Country of Origin",
                    "readOnly": true,
                    "x-globalContext": {
                        "readFrom": "consignee:countryOfOrigin"
                    }
                },
                "declarationType": {
                    "enum": [
                        "EX1"
                    ],
                    "type": "string",
                    "title": "Declaration Type"
                },
                "totalInvoiceValue": {
                    "type": "string",
                    "title": "Total Invoice Value & Currency",
                    "description": "Example: 1,000,000 LKR"
                },
                "countryOfDestination": {
                    "type": "string",
                    "title": "Country of Destination",
                    "readOnly": true,
                    "x-globalContext": {
                        "readFrom": "consignment:destination"
                    }
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "text": "Customs Declaration",
                    "type": "Label"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/declarationType"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/totalInvoiceValue"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/totalPackages"
                },
                {
                    "type": "HorizontalLayout",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/countryOfOrigin",
                            "options": {
                                "readOnly": true
                            }
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/countryOfDestination",
                            "options": {
                                "readOnly": true
                            }
                        }
                    ]
                },
                {
                    "type": "Control",
                    "scope": "#/properties/totalNetWeight"
                }
            ]
        }',
        '1.0',
        TRUE
    ),
    
    -- Phytosanitary Certificate
    (
        '22222222-2222-2222-2222-222222222222',
        'Phytosanitary Certificate',
        'Phytosanitary certificate for plant products export',
        '{
            "type": "object",
            "required": [
                "distinguishingMarks",
                "disinfestationTreatment"
            ],
            "properties": {
                "countryOfOrigin": {
                    "type": "string",
                    "title": "Country of Origin",
                    "readOnly": true,
                    "x-globalContext": {
                        "readFrom": "consignee:countryOfOrigin"
                    }
                },
                "distinguishingMarks": {
                    "type": "string",
                    "title": "Distinguishing Marks",
                    "example": "BWI-UK-LOT01"
                },
                "countryOfDestination": {
                    "type": "string",
                    "title": "Country of Destination",
                    "readOnly": true,
                    "x-globalContext": {
                        "readFrom": "consignment:destination"
                    }
                },
                "disinfestationTreatment": {
                    "type": "string",
                    "title": "Disinfestation Treatment",
                    "example": "Fumigation with Methyl Bromide (CH3Br) at 48g/m³ for 24 hrs"
                },
                "supportingDocuments": {
                    "type": "string",
                    "title": "Supporting Documents",
                    "format": "file"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "text": "Phytosanitary Certificate",
                    "type": "Label"
                },
                {
                    "type": "HorizontalLayout",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/countryOfOrigin",
                            "options": {
                                "readOnly": true
                            }
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/countryOfDestination",
                            "options": {
                                "readOnly": true
                            }
                        }
                    ]
                },
                {
                    "type": "Control",
                    "scope": "#/properties/distinguishingMarks"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/disinfestationTreatment"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/supportingDocuments"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- Health Certificate
    (
        '33333333-3333-3333-3333-333333333333',
        'Health Certificate',
        'Health and safety certificate for food products',
        '{
            "type": "object",
            "required": [
                "productDescription",
                "batchLotNumbers",
                "productionExpiryDates",
                "microbiologicalTestReportId",
                "processingPlantRegistrationNo"
            ],
            "properties": {
                "consigneeName": {
                    "type": "string",
                    "title": "Consignee Name",
                    "readOnly": true,
                    "x-globalContext": {
                        "readFrom": "consignee:consignee_name"
                    }
                },
                "batchLotNumbers": {
                    "type": "string",
                    "title": "Batch / Lot Numbers",
                    "description": "DC-2026-JAN-05"
                },
                "consigneeAddress": {
                    "type": "string",
                    "title": "Consignee Address",
                    "readOnly": true,
                    "x-globalContext": {
                        "readFrom": "consignee:address"
                    }
                },
                "productDescription": {
                    "type": "string",
                    "title": "Product Description",
                    "description": "Organic Desiccated Coconut (Fine Grade)"
                },
                "productionExpiryDates": {
                    "type": "string",
                    "title": "Production & Expiry Dates",
                    "description": "Example: Production: YYYY-MM-DD, Expiry: YYYY-MM-DD"
                },
                "microbiologicalTestReportId": {
                    "type": "string",
                    "title": "Microbiological Test Report ID",
                    "description": "ITI/2026/LAB-9982"
                },
                "processingPlantRegistrationNo": {
                    "type": "string",
                    "title": "Processing Plant Registration No.",
                    "description": "CDA/REG/2025/158"
                },
                "supportingDocuments": {
                    "type": "string",
                    "title": "Supporting Documents",
                    "format": "file"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "text": "Health Certificate",
                    "type": "Label"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/consigneeName",
                    "options": {
                        "readOnly": true
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/consigneeAddress",
                    "options": {
                        "multi": true,
                        "readOnly": true
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/productDescription"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/batchLotNumbers"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/productionExpiryDates"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/microbiologicalTestReportId"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/processingPlantRegistrationNo"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/supportingDocuments"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- General Information
    (
        '44444444-4444-4444-4444-444444444444',
        'General Information',
        'General consignment information form',
        '{
            "type": "object",
            "title": "General Info",
            "required": [
                "consigneeName",
                "consigneeAddress",
                "countryOfOrigin",
                "countryOfDestination"
            ],
            "properties": {
                "consigneeName": {
                    "type": "string",
                    "title": "Consignee Name",
                    "x-globalContext": {
                        "writeTo": "consignee:consignee_name"
                    }
                },
                "countryOfOrigin": {
                    "enum": [
                        "LK"
                    ],
                    "type": "string",
                    "title": "Country of Origin",
                    "x-globalContext": {
                        "writeTo": "consignee:countryOfOrigin"
                    }
                },
                "consigneeAddress": {
                    "type": "string",
                    "title": "Consignee Address",
                    "x-globalContext": {
                        "writeTo": "consignee:address"
                    }
                },
                "countryOfDestination": {
                    "type": "string",
                    "title": "Country of Destination",
                    "x-globalContext": {
                        "writeTo": "consignment:destination"
                    }
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/consigneeName"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/consigneeAddress",
                    "options": {
                        "multi": true
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/countryOfOrigin"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/countryOfDestination"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- OGA Form (Health Certificate) Review
    (
        '95d7e7fe-5be0-43cb-ac71-94bc70d3a01d',
        'OGA Form Review (health certificate)',
        'Form to render review information of health certificate',
        '{
            "type": "object",
            "required": [
                "decision",
                "reviewedAt"
            ],
            "properties": {
                "decision": {
                    "enum": [
                        "APPROVED",
                        "REJECTED"
                    ],
                    "type": "string"
                },
                "reviewedAt": {
                    "type": "string",
                    "format": "date-time"
                },
                "reviewerNotes": {
                    "type": "string"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/decision",
                    "options": {
                        "format": "radio"
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/reviewerNotes",
                    "options": {
                        "multi": true
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/reviewedAt"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- Customs Declaration Request's Response View
    (
        '71e5e73a-ebb2-4750-aaa4-f71087adac43',
        'Customs Declaration Request''s Response View',
        'Response for the cusdec call.',
        '{
            "type": "object",
            "required": [
                "assesmentNo",
                "payment_requirements"
            ],
            "properties": {
                "assesmentNo": {
                    "type": "string",
                    "title": "Assessment No"
                },
                "payment_requirements": {
                    "type": "object",
                    "title": "Payment Requirements",
                    "properties": {
                        "cess": {
                            "type": "number",
                            "title": "Cess"
                        },
                        "total": {
                            "type": "number",
                            "title": "Total"
                        },
                        "export_levy": {
                            "type": "number",
                            "title": "Export Levy"
                        }
                    }
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "label": "Assessment Number",
                    "scope": "#/properties/assesmentNo"
                },
                {
                    "type": "Group",
                    "label": "Payment Requirements",
                    "elements": [
                        {
                            "type": "Control",
                            "scope": "#/properties/payment_requirements/properties/cess"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/payment_requirements/properties/export_levy"
                        },
                        {
                            "type": "Control",
                            "scope": "#/properties/payment_requirements/properties/total"
                        }
                    ]
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- OGA Review (Phytosanitary Certificate) View
    (
        'd0c3b860-635b-4124-8081-d3f421e429cb',
        'OGA Review View (phytosanitary certificate)',
        'Form to display the OGA review response.',
        '{
            "type": "object",
            "required": [
                "decision",
                "phytosanitaryClearance"
            ],
            "properties": {
                "remarks": {
                    "type": "string",
                    "title": "NPQS Remarks"
                },
                "decision": {
                    "type": "string",
                    "oneOf": [
                        {
                            "const": "APPROVED",
                            "title": "Approved"
                        },
                        {
                            "const": "REJECTED",
                            "title": "Rejected"
                        },
                        {
                            "const": "MANUAL_REVIEW",
                            "title": "Manual Inspection Required"
                        }
                    ],
                    "title": "Decision"
                },
                "inspectionReference": {
                    "type": "string",
                    "title": "Inspection / Certificate Reference No"
                },
                "phytosanitaryClearance": {
                    "type": "string",
                    "oneOf": [
                        {
                            "const": "CLEARED",
                            "title": "Cleared for Export"
                        },
                        {
                            "const": "CONDITIONAL",
                            "title": "Cleared with Conditions"
                        },
                        {
                            "const": "REJECTED",
                            "title": "Rejected - Non Compliance"
                        }
                    ],
                    "title": "Phytosanitary Clearance Status"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/decision"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/phytosanitaryClearance"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/inspectionReference"
                },
                {
                    "type": "Control",
                    "scope": "#/properties/remarks",
                    "options": {
                        "multi": true
                    }
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- Manual Inspection Form ---
    (
        'f1a00001-0001-4000-c000-000000000001',
        'Manual Inspection Form (Phytosanitary)',
        'Form for manual inspection tasks when phytosanitary certificate requires manual review',
        '{
            "type": "object",
            "properties": {
                "inspectionDate": {
                    "type": "string",
                    "format": "date",
                    "title": "Inspection Date"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/inspectionDate"
                }
            ]
        }',
        '1.0',
        TRUE
    ),

    -- OGA Review (Manual Inspection) View
    (
        'f1a00001-0001-4000-c000-000000000002',
        'OGA Review (Manual Inspection) View',
        'Form to display OGA review response for manual inspection outcome.',
        '{
            "type": "object",
            "required": [
                "decision",
                "reviewedAt"
            ],
            "properties": {
                "decision": {
                    "enum": [
                        "APPROVED",
                        "REJECTED"
                    ],
                    "type": "string"
                },
                "reviewedAt": {
                    "type": "string",
                    "format": "date-time"
                },
                "reviewerNotes": {
                    "type": "string"
                }
            }
        }',
        '{
            "type": "VerticalLayout",
            "elements": [
                {
                    "type": "Control",
                    "scope": "#/properties/decision",
                    "options": {
                        "format": "radio"
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/reviewerNotes",
                    "options": {
                        "multi": true
                    }
                },
                {
                    "type": "Control",
                    "scope": "#/properties/reviewedAt"
                }
            ]
        }',
        '1.0',
        TRUE
    );