// Package storage owns custodian's SQLite database — the sole source of truth.
// It opens a pure-Go modernc.org/sqlite connection (so the binary stays
// CGO_ENABLED=0 and cross-compiles to ARM64) and runs the embedded migrations
// at startup.
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB is custodian's handle on its SQLite database.
type DB struct {
	*sql.DB
}

// Open connects to the SQLite database at path (a file path, or ":memory:")
// with WAL journalling and foreign keys enabled, then applies all migrations.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) migrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migration (name TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		applied, err := db.migrationApplied(file)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := db.applyMigration(file); err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}
	return nil
}

func (db *DB) migrationApplied(name string) (bool, error) {
	var found string
	err := db.QueryRow(`SELECT name FROM schema_migration WHERE name = ?`, name).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) applyMigration(name string) error {
	statements, err := migrationsFS.ReadFile(name)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(statements)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migration (name) VALUES (?)`, name); err != nil {
		return err
	}
	return tx.Commit()
}
