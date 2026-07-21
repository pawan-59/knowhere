package db

import (
	"database/sql"
	"time"
)

// RowScanner is satisfied by both *sql.Row and *sql.Rows, so store scan helpers
// can accept either.
type RowScanner interface {
	Scan(dest ...any) error
}

// SQLite stores timestamps as RFC3339 TEXT (see schema.sql). These helpers keep
// the Go models working with time.Time while the storage stays text.

// ParseTime parses a required RFC3339 timestamp column.
func ParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseTimePtr parses a nullable timestamp column into *time.Time.
func ParseTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, ns.String)
	if err != nil {
		return nil
	}
	return &t
}

// TimeArg converts a *time.Time to a bind argument (nil or RFC3339 string).
func TimeArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// StrArg converts a *string to a bind argument (database/sql does not
// auto-dereference pointers).
func StrArg(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
