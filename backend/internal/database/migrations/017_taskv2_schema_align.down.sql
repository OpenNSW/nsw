-- Migration: 019_taskv2_schema_align.down.sql
-- Description: Revert task_workflow_tasks to the pre-nsw-task-flow shape.

BEGIN;

DROP TABLE IF EXISTS task_workflow_tasks;

DROP INDEX IF EXISTS idx_task_workflow_tasks_parent_workflow_id;
DROP INDEX IF EXISTS idx_task_workflow_tasks_task_workflow_id;
DROP INDEX IF EXISTS idx_task_workflow_tasks_status;
DROP INDEX IF EXISTS idx_task_workflow_tasks_subtask_node_id;
DROP INDEX IF EXISTS idx_task_workflow_tasks_parent_node_id;

COMMIT;