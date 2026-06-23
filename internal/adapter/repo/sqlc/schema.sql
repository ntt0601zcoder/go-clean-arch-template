-- schema.sql is the input for `sqlc generate`. It mirrors the golang-migrate
-- migration in migrations/. Keep the two in sync (Makefile target `make sqlc`).
CREATE TABLE accounts (
    id            TEXT PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    first_name    TEXT        NOT NULL DEFAULT '',
    last_name     TEXT        NOT NULL DEFAULT '',
    password_hash TEXT        NOT NULL DEFAULT '',
    status        INT         NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_accounts_status ON accounts (status);
