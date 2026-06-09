package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func runMigrate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: nats-admin migrate <up|down|version> <dsn>")
	}

	dsn := args[1]
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrate fs: %w", err)
	}
	src, err := iofs.New(sub, ".")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	direction := args[0]
	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			return fmt.Errorf("version: %w", verr)
		}
		fmt.Printf("version: %d dirty: %v\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate %s: %w", direction, err)
	}
	fmt.Println("migrate:", direction, "OK")
	return nil
}
