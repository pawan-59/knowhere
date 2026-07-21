-- Central-Devtron schema (SQLite). Applied idempotently on server startup.
-- Timestamps are stored as RFC3339 TEXT (UTC). See internal/db/helpers.go.

-- ── Auth ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,               -- bcrypt
    name          TEXT,
    role          TEXT    NOT NULL DEFAULT 'admin',
    disabled      INTEGER NOT NULL DEFAULT 0,      -- 0/1
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── License monitoring ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS licenses (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    customer        TEXT    NOT NULL,
    installation    TEXT    NOT NULL,
    edition         TEXT    NOT NULL DEFAULT 'enterprise',
    seats           INTEGER NOT NULL DEFAULT 0,
    seats_used      INTEGER NOT NULL DEFAULT 0,
    status          TEXT    NOT NULL DEFAULT 'active',   -- active | expired | trial | suspended
    issued_at       TEXT,
    expires_at      TEXT,
    devtron_version TEXT,
    notes           TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE (customer, installation)
);

CREATE INDEX IF NOT EXISTS idx_licenses_status  ON licenses (status);
CREATE INDEX IF NOT EXISTS idx_licenses_expires ON licenses (expires_at);

-- ── Onboarding tracking ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS onboardings (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    customer      TEXT    NOT NULL UNIQUE,
    owner         TEXT,
    stage         TEXT    NOT NULL DEFAULT 'Discovery Call',  -- pipeline stage (see frontend STAGES)
    status        TEXT    NOT NULL DEFAULT 'on_track',   -- on_track | at_risk | blocked | completed
    progress      INTEGER NOT NULL DEFAULT 0,            -- 0-100
    started_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    target_date   TEXT,
    completed_at  TEXT,
    notes         TEXT,
    blocked_reason TEXT,                                 -- where it's blocked (for "Blocked On …" stages)
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_onboardings_stage  ON onboardings (stage);
CREATE INDEX IF NOT EXISTS idx_onboardings_status ON onboardings (status);
