-- Description: Roll back workflow seed data.

DELETE FROM workflow_template_map
WHERE id = 'fcau-wf-map-0001';

DELETE FROM hs_codes
WHERE id = 'fcau-hs-code-0001';

DELETE FROM workflow_template_v2
WHERE id = 'fcau-health-certificate-reg';
