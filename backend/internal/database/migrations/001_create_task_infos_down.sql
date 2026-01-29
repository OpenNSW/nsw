-- Migration: 001_create_task_infos_down.sql
-- Description: Rollback task_infos table creation
-- Created: 2026-01-29

-- Drop task_infos table
DROP TABLE IF EXISTS task_infos;
