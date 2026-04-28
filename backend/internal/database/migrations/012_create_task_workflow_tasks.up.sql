BEGIN;

CREATE TABLE IF NOT EXISTS task_workflow_tasks (
    task_id text NOT NULL PRIMARY KEY,
    macro_workflow_id text NOT NULL,
    task_template_id text NOT NULL,
    state varchar(50) NOT NULL,
    data jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE task_workflow_tasks IS 'Durable records for tasks executed inside task workflows';
COMMENT ON COLUMN task_workflow_tasks.task_id IS 'Unique task instance ID';
COMMENT ON COLUMN task_workflow_tasks.macro_workflow_id IS 'Outer workflow instance that owns this task workflow task';
COMMENT ON COLUMN task_workflow_tasks.task_template_id IS 'Template used to execute/render this task';
COMMENT ON COLUMN task_workflow_tasks.state IS 'Current task state';
COMMENT ON COLUMN task_workflow_tasks.data IS 'JSON payload used to render the task';

CREATE INDEX IF NOT EXISTS idx_task_workflow_tasks_macro_workflow_id ON task_workflow_tasks (macro_workflow_id);
CREATE INDEX IF NOT EXISTS idx_task_workflow_tasks_task_template_id ON task_workflow_tasks (task_template_id);
CREATE INDEX IF NOT EXISTS idx_task_workflow_tasks_state ON task_workflow_tasks (state);

COMMIT;
