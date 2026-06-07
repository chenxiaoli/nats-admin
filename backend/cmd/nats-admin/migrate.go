package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: nats-admin migrate <up|down|version> <dsn>")
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir, _ = os.Getwd()
		dir = filepath.Join(dir, "internal", "db", "migrations")
	}
	src := "file://" + dir
	dsn := args[1]

	m, err := migrate.New(src, dsn)
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
