package database

import (
	"database/sql"
	"fmt"
	"log"
)

// migrate runs all SQL statements that create the application schema.
// Each statement uses IF NOT EXISTS so it is safe to run on every startup.
//
// Schema design notes:
//   - UUIDs (TEXT) are used for primary keys instead of auto-increment integers.
//     This avoids leaking row counts in URLs and makes IDs safe to use in URLs.
//   - completed is stored as BOOLEAN (SQLite stores it as 0/1 integer).
//   - priority is constrained to 'low'|'medium'|'high' via a CHECK constraint.
//   - updated_at uses a trigger to auto-update on every row modification.
func migrate(db *sql.DB) error {
	statements := []struct {
		name string
		sql  string
	}{
		{
			name: "create users table",
			sql: `
			CREATE TABLE IF NOT EXISTS users (
				id           TEXT PRIMARY KEY,
				email        TEXT UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "create tasks table",
			sql: `
			CREATE TABLE IF NOT EXISTS tasks (
				id                     TEXT PRIMARY KEY,
				user_id                TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				title                  TEXT NOT NULL,
				description            TEXT DEFAULT '',
				completed              BOOLEAN NOT NULL DEFAULT 0,
				priority               TEXT NOT NULL DEFAULT 'medium'
				                       CHECK(priority IN ('low','medium','high')),
				due_date               DATETIME,
				recurring              TEXT NOT NULL DEFAULT 'none'
				                       CHECK(recurring IN ('none','daily','weekly','monthly')),
				timer_duration_minutes INTEGER,
				created_at             DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at             DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "create updated_at trigger",
			sql: `
			CREATE TRIGGER IF NOT EXISTS tasks_updated_at
			AFTER UPDATE ON tasks
			BEGIN
				UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END`,
		},
		{
			name: "create tasks user_id index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id)`,
		},
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt.sql); err != nil {
			return fmt.Errorf("migration %q failed: %w", stmt.name, err)
		}
		log.Printf("  ✔ migration: %s", stmt.name)
	}

	// Additive column migrations — ALTER TABLE ADD COLUMN is safe to re-run;
	// SQLite returns an error if the column already exists, which we ignore.
	addColumns := []struct{ name, sql string }{
		{"add recurring column",              `ALTER TABLE tasks ADD COLUMN recurring TEXT NOT NULL DEFAULT 'none'`},
		{"add timer_duration_minutes column", `ALTER TABLE tasks ADD COLUMN timer_duration_minutes INTEGER`},
	}
	for _, col := range addColumns {
		if _, err := db.Exec(col.sql); err != nil {
			// "duplicate column name" means the column already exists — that's fine.
			if !isDuplicateColumnError(err) {
				return fmt.Errorf("migration %q failed: %w", col.name, err)
			}
		}
		log.Printf("  ✔ migration: %s", col.name)
	}

	return nil
}

// isDuplicateColumnError returns true when SQLite complains that a column
// being added via ALTER TABLE already exists.
func isDuplicateColumnError(err error) bool {
	return err != nil && (
		// modernc.org/sqlite surfaces this message
		contains(err.Error(), "duplicate column name") ||
		contains(err.Error(), "already exists"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
