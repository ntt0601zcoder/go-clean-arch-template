// Package migrate runs database migrations and exits. It deliberately does NOT
// use the full FX graph (no mongo/redis/kafka): it opens only a PostgreSQL
// connection so `migrate` stays fast and works when only Postgres is up.
package migrate

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/config"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/db"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/logger"
	"github.com/ntt0601zcoder/go-clean-arch-template/migrations"
)

// Run applies the embedded migrations to PostgreSQL.
func Run() error {
	cfg := config.GetConfig()
	log := logger.NewSlogger("migrate", cfg.App.LogLevel)

	sqlDB, err := sql.Open("pgx", cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer sqlDB.Close()

	if err := db.RunMigrations(sqlDB, migrations.FS, cfg.Postgres.MigrateVersion); err != nil {
		return err
	}
	log.Info("migrations applied")
	return nil
}
