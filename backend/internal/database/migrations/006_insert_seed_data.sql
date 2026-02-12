INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('95d7e7fe-5be0-43cb-ac71-94bc70d3a01d', 'OGA Form Review', 'Form to render review information of phytosanitary certificate',
 '{"type": "object", "required": ["decision", "reviewedAt"], "properties": {"decision": {"enum": ["APPROVED", "REJECTED", "NEEDS_MORE_INFO"], "type": "string"}, "reviewedAt": {"type": "string", "format": "date-time"}, "reviewerNotes": {"type": "string"}}}',
 '{"type": "VerticalLayout", "elements": [{"type": "Control", "scope": "#/properties/decision", "options": {"format": "radio"}}, {"type": "Control", "scope": "#/properties/reviewerNotes", "options": {"multi": true}}, {"type": "Control", "scope": "#/properties/reviewedAt"}]}',
    '1.0', true);

UPDATE workflow_node_templates SET config = '{"agency": "NPQS", "formId": "22222222-2222-2222-2222-222222222222", "service": "plant-quarantine-phytosanitary", "callback": {"response": {"display": {"formId": "95d7e7fe-5be0-43cb-ac71-94bc70d3a01d"}, "mapping": {"reviewedAt": "gi:phytosanitary:meta:reviewedAt", "reviewerNotes": "gi:phytosanitary:meta:reviewNotes"}}}, "submission": {"url": "http://localhost:8081/api/oga/inject"}}'::jsonb
WHERE id = 'c0000003-0003-0003-0003-000000000003';

UPDATE workflow_node_templates SET config = '{"agency": "EDB", "formId": "33333333-3333-3333-3333-333333333333", "service": "food-control-administration-unit", "callback": {"response": {"display": {"formId": "95d7e7fe-5be0-43cb-ac71-94bc70d3a01d"}}}, "submission": {"url": "http://localhost:8082/api/oga/inject"}}'::jsonb
WHERE id = 'c0000003-0003-0003-0003-000000000004';
