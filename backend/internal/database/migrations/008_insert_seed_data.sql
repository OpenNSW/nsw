INSERT INTO forms (id, name, description, schema, ui_schema, version, active, created_at, updated_at)
VALUES ('39498bef-aef6-45d5-bc94-e13a52ad3dbf',
        'Health Certificate Review',
        'Review from NPQS.',
        '{"type": "object", "required": ["decision", "foodSafetyClearance"], "properties": {"remarks": {"type": "string", "title": "FCAU Remarks"}, "decision": {"type": "string", "oneOf": [{"const": "APPROVED", "title": "Approved"}, {"const": "REJECTED", "title": "Rejected"}], "title": "Decision"},
    "labReportReference": {"type": "string", "title": "Laboratory Report Reference No"}, "foodSafetyClearance": {"type": "string", "oneOf": [{"const": "COMPLIANT", "title": "Compliant - Approved for Export"}, {"const": "MINOR_NON_COMPLIANCE", "title": "Minor Non-Compliance"}, {"const": "MAJOR_NON_COMPLIANCE",
    "title": "Major Non-Compliance"}], "title": "Food Safety Compliance Status"}}}',
        '{"type": "VerticalLayout", "elements": [{"type": "Control", "scope": "#/properties/decision"}, {"type": "Control", "scope": "#/properties/foodSafetyClearance"}, {"type": "Control", "scope": "#/properties/labReportReference"}, {"type": "Control", "scope": "#/properties/remarks", "options": {"multi":
    true}}]}',
        1.0,
        true,
        '2026-02-13 03:33:35.072026 +00:00',
        '2026-02-13 03:33:35.072026 +00:00'),
       ('d0c3b860-635b-4124-8081-d3f421e429cb',
        'Phytosanitary Certificate Review',
        'Review from FCAU.',
        '{"type": "object", "required": ["decision", "phytosanitaryClearance"], "properties": {"remarks": {"type": "string", "title": "NPQS Remarks"}, "decision": {"type": "string", "oneOf": [{"const": "APPROVED", "title": "Approved"}, {"const": "REJECTED", "title": "Rejected"}], "title": "Decision"},
    "inspectionReference": {"type": "string", "title": "Inspection / Certificate Reference No"}, "phytosanitaryClearance": {"type": "string", "oneOf": [{"const": "CLEARED", "title": "Cleared for Export"}, {"const": "CONDITIONAL", "title": "Cleared with Conditions"}, {"const": "REJECTED", "title": "Rejected - Non
    Compliance"}], "title": "Phytosanitary Clearance Status"}}}',
        '{"type": "VerticalLayout", "elements": [{"type": "Control", "scope": "#/properties/decision"}, {"type": "Control", "scope": "#/properties/phytosanitaryClearance"}, {"type": "Control", "scope": "#/properties/inspectionReference"}, {"type": "Control", "scope": "#/properties/remarks", "options": {"multi":
    true}}]}',
        1.0,
        true,
        '2026-02-13 03:35:20.950159 +00:00',
        '2026-02-13 03:35:20.950159 +00:00');


UPDATE workflow_node_templates
SET config = '{
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
        }
    },
    "submission": {
        "url": "http://localhost:8081/api/oga/inject",
        "request": {
            "meta": {
                "type": "consignment",
                "verificationId": "moa:npqs:phytosanitary:001"
            }
        }
    }
}'
WHERE id = 'c0000003-0003-0003-0003-000000000003';

UPDATE workflow_node_templates
SET config = '{
    "agency": "FCAU",
    "formId": "33333333-3333-3333-3333-333333333333",
    "service": "food-control-administration-unit",
    "callback": {
        "response": {
            "display": {
                "formId": "39498bef-aef6-45d5-bc94-e13a52ad3dbf"
            }
        }
    },
    "submission": {
        "url": "http://localhost:8082/api/oga/inject",
        "request": {
            "meta": {
                "type": "consignment",
                "verificationId": "moh:fcau:health_cert:001"
            }
        }
    }
}'
WHERE id = 'c0000003-0003-0003-0003-000000000004';