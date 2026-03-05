-- Migration: 001_add_cha_entity_down.sql

ALTER TABLE consignments DROP COLUMN cha_id;
DROP INDEX IF EXISTS idx_consignments_cha_id;
DROP TABLE IF EXISTS customs_house_agents;
