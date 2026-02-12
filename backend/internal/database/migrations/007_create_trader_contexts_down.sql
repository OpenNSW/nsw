-- Rollback: Drop trader_contexts table
-- This migration reverses the creation of the trader_contexts table

DROP TABLE IF EXISTS trader_contexts CASCADE;
