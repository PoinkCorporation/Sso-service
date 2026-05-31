package main

import (
	"errors"
	"flag"
	"fmt"
	"sso/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		migrationsPath  string
		migrationsTable string
	)

	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&migrationsTable, "migrations-table", "schema_migrations", "name of migrations table")

	cfg := config.MustLoad()

	if migrationsPath == "" {
		panic("migrations-path is required")
	}

	dbURL := cfg.StoragePath + "?sslmode=disable"

	m, err := migrate.New(
		"file://"+migrationsPath,
		dbURL,
	)
	if err != nil {
		panic(err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}
		panic(err)
	}

	fmt.Println("migrations applied successfully")
}
