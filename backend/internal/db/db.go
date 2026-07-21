// Package db manages the embedded SQLite database and schema migration.
//
// SQLite (via the pure-Go modernc.org/sqlite driver — no CGO) keeps the whole
// app in a single container: the data lives in one file, which on Kubernetes is
// placed on a PersistentVolume so it survives pod rotation.
package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Connect opens the SQLite database at path, tunes pragmas for durability and
// concurrency, and applies the schema.
func Connect(ctx context.Context, path string) (*sql.DB, error) {
	// busy_timeout makes writers wait rather than fail when the file is locked.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", filepath.Clean(path))

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite has a single writer. Serialize connections to avoid "database is
	// locked" under concurrent writes — trivial cost at dashboard scale.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	// WAL improves read/write concurrency and crash resilience.
	if _, err := sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}

	if _, err := sqlDB.ExecContext(ctx, schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	// Additive migrations for databases created before a column existed.
	// CREATE TABLE IF NOT EXISTS won't alter an existing table, so backfill here.
	if err := ensureColumn(ctx, sqlDB, "onboardings", "blocked_reason", "TEXT"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate onboardings.blocked_reason: %w", err)
	}

	return sqlDB, nil
}

// ensureColumn adds a column to a table if it is not already present, so
// migrations stay idempotent (SQLite has no ADD COLUMN IF NOT EXISTS). Table and
// column names are compile-time constants — never user input — so the string
// interpolation here is safe.
func ensureColumn(ctx context.Context, sqlDB *sql.DB, table, column, typ string) error {
	rows, err := sqlDB.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil // already present
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = sqlDB.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+column+" "+typ)
	return err
}
