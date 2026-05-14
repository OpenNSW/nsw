-- ============================================================================
-- Migration: 019_taskv2_schema_align.up.sql
-- Purpose: Align task_workflow_tasks with the nsw-task-flow store.TaskRecord
--          shape now that nsw-task-flow has replaced the legacy TaskManager.
--
-- Drops and recreates the table because earlier columns (macro_workflow_id,
-- task_template_id, state) don't map cleanly to TaskRecord and the table is
-- empty in development. Production data, if any, would need a real ALTER.
-- ============================================================================

BEGIN;

DROP TABLE IF EXISTS task_workflow_tasks;

CREATE TABLE task_workflow_tasks (
    task_id                 text        NOT NULL PRIMARY KEY,
    task_type               text        NOT NULL DEFAULT '',
    user_form_id            text        NOT NULL DEFAULT '',
    reviewer_form_id        text        NOT NULL DEFAULT '',
    status                  varchar(50) NOT NULL,

    -- Parent (macro) workflow coordinates
    parent_workflow_id      text        NOT NULL DEFAULT '',
    parent_run_id           text        NOT NULL DEFAULT '',
    parent_node_id          text        NOT NULL DEFAULT '',

    -- Task (sub-)workflow coordinates
    task_workflow_id        text        NOT NULL DEFAULT '',
    task_run_id             text        NOT NULL DEFAULT '',
    subtask_node_id         text        NOT NULL DEFAULT '',
    active_task_template_id text        NOT NULL DEFAULT '',

    data        jsonb                    NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamp with time zone NOT NULL DEFAULT now(),
    updated_at  timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_workflow_tasks_parent_workflow_id ON task_workflow_tasks (parent_workflow_id);
CREATE INDEX idx_task_workflow_tasks_task_workflow_id   ON task_workflow_tasks (task_workflow_id);
CREATE INDEX idx_task_workflow_tasks_status             ON task_workflow_tasks (status);
CREATE INDEX idx_task_workflow_tasks_subtask_node_id    ON task_workflow_tasks (subtask_node_id);  
CREATE INDEX idx_task_workflow_tasks_parent_node_id     ON task_workflow_tasks (parent_node_id);  


COMMENT ON TABLE  task_workflow_tasks IS 'TaskRecords managed by nsw-task-flow orchestrator (1 row per task instance)';
COMMENT ON COLUMN task_workflow_tasks.task_id            IS 'Unique task instance identifier (task-XXXXXXXX)';
COMMENT ON COLUMN task_workflow_tasks.task_type          IS 'Task type/category (e.g. APPLICATION, WAIT_FOR_EVENT, PAYMENT, COMPOSITE)';
COMMENT ON COLUMN task_workflow_tasks.status             IS 'Status string consumed by the UI (PENDING_USER, QUEUED_EXTERNALLY, COMPLETED, ...)';
COMMENT ON COLUMN task_workflow_tasks.parent_workflow_id IS 'Workflow ID of the macro/parent workflow that owns this task';
COMMENT ON COLUMN task_workflow_tasks.task_workflow_id   IS 'Workflow ID of the dedicated sub-workflow executing this task';
COMMENT ON COLUMN task_workflow_tasks.subtask_node_id    IS 'Currently active sub-task node within task_workflow_id';
COMMENT ON COLUMN task_workflow_tasks.data               IS 'Namespaced execution variables for this task (form payloads, mappings, etc.)';

COMMIT;