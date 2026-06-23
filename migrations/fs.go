// Package migrations embeds the SQL migration files so they ship inside the
// binary and can be applied at boot (config APP/POSTGRES_MIGRATE) or via the
// `migrate` subcommand. They are the source of truth for the PostgreSQL schema;
// keep internal/adapter/repo/sqlc/schema.sql in sync.
package migrations

import "embed"

// FS holds the embedded *.sql migration files (golang-migrate iofs source).
//
//go:embed *.sql
var FS embed.FS
