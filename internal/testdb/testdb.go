// Package testdb gives each test package its own freshly migrated database.
package testdb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Create makes a migrated database named after TEST_DATABASE_URL's plus
// suffix. go test runs packages in parallel, so each package needs its own
// suffix or they drop each other's databases. Call the returned func to drop it.
func Create(suffix string) (*pgxpool.Pool, func()) {
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		log.Fatal("TEST_DATABASE_URL must be set")
	}
	dbName := GetDBName(baseURL) + "_" + suffix
	dbURL := SwapDBName(baseURL, dbName)

	ctx := context.Background()

	pgDB := connect(ctx, SwapDBName(baseURL, "postgres"))
	pgDB.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if _, err := pgDB.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		log.Fatalf("could not create test database: %v", err)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	migrateURL := strings.Replace(dbURL, "postgres://", "pgx5://", 1)
	mig, err := migrate.New("file://"+migrationsDir, migrateURL)
	if err != nil {
		log.Fatalf("could not create migrate instance: %v", err)
	}
	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("could not migrate test db: %v", err)
	}
	mig.Close()

	pool := connect(ctx, dbURL)

	drop := func() {
		pool.Close()
		// Postgres refuses to drop a DB with active connections -- force evictions first
		pgDB.Exec(ctx, `
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = $1 AND pid <> pg_backend_pid()
		`, dbName)
		pgDB.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		pgDB.Close()
	}
	return pool, drop
}

func connect(ctx context.Context, dbURL string) *pgxpool.Pool {
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("error establishing test database connection: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("cannot ping test database %s: %v", dbURL, err)
	}
	return db
}

func GetDBName(dbURL string) string {
	u, _ := url.Parse(dbURL)
	return u.Path[1:]
}

func SwapDBName(oldDB, newDB string) string {
	u, _ := url.Parse(oldDB)
	u.Path = "/" + newDB
	return u.String()
}
