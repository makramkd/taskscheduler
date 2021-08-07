-- All the migrations in order to have a working database.

\ir migrations/001_create_db.sql

\connect taskdb

\ir migrations/002_create_tasks.sql
