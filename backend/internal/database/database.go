// Package database manages the SQLite connection and schema migrations.
//
// WHY SQLite?
//   - Zero infrastructure — the DB lives in a single file alongside the binary.
//     Windows and cross-compiles to Linux for Docker without toolchain headaches.
//
// WHY auto-migrations at startup?
//   - On first run the tables are created.
//   - On subsequent runs the IF NOT EXISTS clauses are no-ops.
package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite" // Register the "sqlite" driver with database/sql
)

// Open opens (or creates) the SQLite database at the given file path,
// configures the connection pool, and runs all schema migrations.
// It returns the ready-to-use *sql.DB or an error.
func Open(dsn string) (*sql.DB, error) {
	// "sqlite" here refers to the driver name registered by modernc.org/sqlite.
	// The DSN is simply the file path, e.g. "./todos.db".
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", dsn, err)
	}

	// SQLite supports only one writer at a time.
	// Setting MaxOpenConns=1 prevents "database is locked" errors.
	db.SetMaxOpenConns(1)

	// Verify the connection is actually working before returning.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Enable WAL (Write-Ahead Logging) mode for better concurrent read performance.
	// Even with MaxOpenConns=1 for writes, reads can still happen concurrently.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	// Enforce foreign key constraints (SQLite disables them by default!).
	// This ensures tasks.user_id references a real users.id.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	log.Printf("Database connected: %s", dsn)

	// Run schema migrations to create tables if they don't exist.
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}
