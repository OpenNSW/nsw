-- Migration: 009_insert_cha_entity.down.sql
-- Description: Roll back CHA entity seed data.
-- Clear operational data that depends on these entities
DELETE FROM consignments;

DELETE FROM customs_house_agents 
WHERE id IN (
    'a1b2c3d4-0001-4000-8000-000000000001',
    'a1b2c3d4-0002-4000-8000-000000000002',
    'a1b2c3d4-0003-4000-8000-000000000003',
    'a1b2c3d4-0004-4000-8000-000000000004',
    'a1b2c3d4-0005-4000-8000-000000000005'
);
