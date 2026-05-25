CREATE TABLE task_records_v2 (
    task_id                 TEXT PRIMARY KEY,
    task_type               TEXT,
    state                   TEXT,
    render_config           JSONB,
    parent_workflow_id      TEXT,
    parent_run_id           TEXT,
    parent_node_id          TEXT,
    task_workflow_id        TEXT,
    task_run_id             TEXT,
    subtask_node_id         TEXT,
    active_task_template_id TEXT,
    data                    JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_records_v2_parent_workflow_id ON task_records_v2(parent_workflow_id);
CREATE INDEX idx_task_records_v2_task_workflow_id ON task_records_v2(task_workflow_id);