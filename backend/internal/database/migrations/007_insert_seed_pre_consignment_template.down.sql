-- Migration: 007_insert_seed_pre_consignment_template.down.sql
-- Description: Roll back pre-consignment template seed data.

DELETE FROM pre_consignment_templates 
WHERE id IN (
    '0c000004-0001-0001-0001-000000000001',
    '0c000004-0001-0001-0001-000000000002'
);
