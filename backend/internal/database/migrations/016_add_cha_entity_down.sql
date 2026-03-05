-- Migration: 016_add_cha_entity.sql

ALTER TABLE consignments DROP COLUMN cha_id;
DROP TABLE clearing_house_agents;
