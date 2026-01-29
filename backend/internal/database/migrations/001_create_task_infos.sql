-- Migration: 001_create_task_infos.sql
-- Description: Create task_infos table for task execution tracking
-- Created: 2026-01-29

-- ============================================================================
-- Table: task_infos
-- Description: Task executable information and state management
-- ============================================================================
CREATE TABLE IF NOT EXISTS task_infos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    step_id VARCHAR(50) NOT NULL,
    consignment_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('SIMPLE_FORM', 'WAIT_FOR_EVENT')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('LOCKED', 'READY', 'IN_PROGRESS', 'COMPLETED', 'REJECTED')),
    command_set JSONB,
    global_context JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_task_infos_consignment_id ON task_infos(consignment_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_step_id ON task_infos(step_id);
CREATE INDEX IF NOT EXISTS idx_task_infos_status ON task_infos(status);
CREATE INDEX IF NOT EXISTS idx_task_infos_type ON task_infos(type);

-- GIN indexes for JSONB columns for faster JSON queries
CREATE INDEX IF NOT EXISTS idx_task_infos_command_set ON task_infos USING GIN (command_set);
CREATE INDEX IF NOT EXISTS idx_task_infos_global_context ON task_infos USING GIN (global_context);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_task_infos_consignment_status ON task_infos(consignment_id, status);

-- ============================================================================
-- Comments for documentation
-- ============================================================================
COMMENT ON TABLE task_infos IS 'Task executable information and state management for the ExecutionUnit Manager';
COMMENT ON COLUMN task_infos.step_id IS 'Unique identifier of the step within the workflow template';
COMMENT ON COLUMN task_infos.command_set IS 'JSONB configuration specific to the task type';
COMMENT ON COLUMN task_infos.global_context IS 'JSONB global context shared across task execution';
