BEGIN;

CREATE TABLE tasks(
    id UUID PRIMARY KEY,
    command TEXT NOT NULL,
    frequency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    servers TEXT[] NOT NULL
);

CREATE TABLE task_outputs(
    task_id UUID NOT NULL,
    outputs JSONB NOT NULL,
    completed_at TIMESTAMPTZ
);

INSERT INTO schema_migrations (version) VALUES ('002_create_tasks');

COMMIT;
