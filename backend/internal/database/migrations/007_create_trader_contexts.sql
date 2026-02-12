-- Migration: Create trader_contexts table
-- Purpose: Store trader context information for authentication and authorization
-- This table holds metadata about traders that can be used across the system

CREATE TABLE trader_contexts (
    trader_id VARCHAR(100) PRIMARY KEY NOT NULL,
    trader_context JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Add comment for documentation
COMMENT ON TABLE trader_contexts IS 'Stores trader context information including metadata in JSON format. This table is used for trader identification and authorization.';
COMMENT ON COLUMN trader_contexts.trader_id IS 'Unique trader identifier (e.g., TRADER-001)';
COMMENT ON COLUMN trader_contexts.trader_context IS 'JSONB field containing trader metadata and context information';
