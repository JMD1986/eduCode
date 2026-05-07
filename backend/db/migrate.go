package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // register driver for database.Open (used by NewWithSourceInstance)
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq" // register database/sql "postgres" driver (used by migrate's postgres driver)
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func openMigrate(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return m, nil
}

// MigrateUp applies all pending migrations.
func MigrateUp(databaseURL string) (retErr error) {
	m, err := openMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if _, closeErr := m.Close(); retErr == nil {
			retErr = closeErr
		}
	}()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// MigrateDown rolls back one migration step. It returns migrate.ErrNoChange if already at version 0.
func MigrateDown(databaseURL string) (retErr error) {
	m, err := openMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if _, closeErr := m.Close(); retErr == nil {
			retErr = closeErr
		}
	}()
	if err := m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return migrate.ErrNoChange
		}
		return err
	}
	return nil
}

// MigrateDownAll rolls back until no migrations are applied.
func MigrateDownAll(databaseURL string) error {
	for {
		err := MigrateDown(databaseURL)
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
