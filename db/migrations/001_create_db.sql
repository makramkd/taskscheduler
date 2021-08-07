CREATE ROLE tasksched_rw LOGIN PASSWORD 'devsecret';
CREATE ROLE tasksched_ro LOGIN PASSWORD 'devsecret';
CREATE ROLE admin_user LOGIN PASSWORD 'admin';

CREATE DATABASE taskdb WITH ENCODING = 'UTF8';

\connect taskdb

GRANT CONNECT, TEMP ON DATABASE taskdb TO tasksched_rw;
GRANT CONNECT, TEMP ON DATABASE taskdb TO tasksched_ro;

ALTER DEFAULT PRIVILEGES IN SCHEMA "public" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tasksched_rw;
ALTER DEFAULT PRIVILEGES IN SCHEMA "public" GRANT USAGE, SELECT ON SEQUENCES TO tasksched_rw;

ALTER DEFAULT PRIVILEGES IN SCHEMA "public" GRANT SELECT ON TABLES TO tasksched_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA "public" GRANT SELECT ON SEQUENCES TO tasksched_ro;

-- Add a record to this table whenever a schema migration is performed.
CREATE TABLE schema_migrations (
    version text,
    timestamp TIMESTAMP WITH TIME ZONE default NOW(),

    PRIMARY KEY (version)
);
