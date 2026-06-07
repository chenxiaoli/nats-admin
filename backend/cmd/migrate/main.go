// Command migrate applies or rolls back the SQL files under
// internal/db/migrations. Usage: migrate <up|down> <dsn>
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

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|version|force V> <dsn>")
		os.Exit(2)
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir, _ = os.Getwd()
		dir = filepath.Join(dir, "internal", "db", "migrations")
	}
	src := "file://" + dir
	dsn := os.Args[2]

	m, err := migrate.New(src, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate init: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	switch os.Args[1] {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			fmt.Fprintf(os.Stderr, "version: %v\n", verr)
			os.Exit(1)
		}
		fmt.Printf("version: %d dirty: %v\n", v, dirty)
		return
	default:
		fmt.Fprintln(os.Stderr, "unknown direction:", os.Args[1])
		os.Exit(2)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Fprintf(os.Stderr, "migrate %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	fmt.Println("migrate:", os.Args[1], "OK")
}
