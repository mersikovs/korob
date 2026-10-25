// Package database provides utilities for applying PostgreSQL schema migrations
// using golang-migrate.
package database

import (
	"github.com/golang-migrate/migrate/v4"
	// Register the PostgreSQL driver for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Register the file source driver for golang-migrate to read migrations from the filesystem.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateUp applies all migrations
func MigrateUp(dsn, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil

}
