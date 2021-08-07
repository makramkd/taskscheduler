BEGIN;

CREATE TABLE task_outputs(
    task_id UUID NOT NULL,
    outputs JSONB NOT NULL,
    completed_at TIMESTAMPTZ
);

CREATE INDEX task_outputs_completed_at_idx ON task_outputs (completed_at);

INSERT INTO schema_migrations (version) VALUES ('002_create_tasks');

COMMIT;
