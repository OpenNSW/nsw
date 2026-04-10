BEGIN;
-- Migration: 007_insert_seed_pre_consignment_template.down.sql
-- Description: Roll back pre-consignment template seed data.
-- Clear operational data that depends on these templates
DELETE FROM pre_consignments;

DELETE FROM pre_consignment_templates 
WHERE id IN (
    '0c000004-0001-0001-0001-000000000001',
    '0c000004-0001-0001-0001-000000000002'
);

DELETE FROM workflow_templates
WHERE id IN (
    'e0000002-0001-0001-0001-000000000004',
    'e0000002-0001-0001-0001-000000000005'
);

DELETE FROM workflow_node_templates
WHERE id IN (
    'd0000002-0001-0001-0001-000000000004',
    'd0000002-0001-0001-0001-000000000005'
);

DELETE FROM forms
WHERE id IN (
    'f0000002-0001-0001-0001-000000000004',
    'f0000002-0001-0001-0001-000000000005'
);
COMMIT;
