package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies the embedded SQL migrations to the PostgreSQL database
// behind sqlDB. version==0 migrates to latest; otherwise it migrates to the exact
// version. ErrNoChange is treated as success (already up to date).
func RunMigrations(sqlDB *sql.DB, fsys fs.FS, version uint) error {
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("migrate: iofs source: %w", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate: postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: new instance: %w", err)
	}

	if version > 0 {
		err = m.Migrate(version)
	} else {
		err = m.Up()
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: apply: %w", err)
	}
	return nil
}
