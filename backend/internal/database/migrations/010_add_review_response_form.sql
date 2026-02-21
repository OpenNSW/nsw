UPDATE workflow_node_templates
SET config = '{"formId": "11111111-1111-1111-1111-111111111111", "submission": {"url": "https://7b0eb5f0-1ee3-4a0c-8946-82a893cb60c2.mock.pstmn.io/api/cusdec", "response": {"display": {"formId": "71e5e73a-ebb2-4750-aaa4-f71087adac43"}, "mapping": {"assesmentNo": "gi:cusdec:assesmentNo"}}}}'
WHERE id = 'c0000003-0003-0003-0003-000000000002';


INSERT INTO forms (id, name, description, schema, ui_schema)
VALUES ('71e5e73a-ebb2-4750-aaa4-f71087adac43',
        'Customs Declaration Request''s Response View',
        'Response for the cusdec call.',
        '{"type": "object", "required": ["assesmentNo", "payment_requirements"], "properties": {"assesmentNo": {"type": "string", "title": "Assessment No"}, "payment_requirements": {"type": "object", "title": "Payment Requirements", "properties": {"cess": {"type": "number", "title": "Cess"}, "total": {"type": "number", "title": "Total"}, "export_levy": {"type": "number", "title": "Export Levy"}}}}}',
        '{"type": "VerticalLayout", "elements": [{"type": "Control", "label": "Assessment Number", "scope": "#/properties/assesmentNo"}, {"type": "Group", "label": "Payment Requirements", "elements": [{"type": "Control", "scope": "#/properties/payment_requirements/properties/cess"}, {"type": "Control", "scope": "#/properties/payment_requirements/properties/export_levy"}, {"type": "Control", "scope": "#/properties/payment_requirements/properties/total"}]}]}');