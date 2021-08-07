FROM postgres:12

COPY ./migrations/*.sql /docker-entrypoint-initdb.d/migrations/
COPY ./schema.sql /docker-entrypoint-initdb.d/1_schema.sql
