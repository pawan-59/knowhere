-- Knowhere schema (SQLite). Applied idempotently on server startup.
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

-- ── Onboarding / POC tracking ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS onboardings (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    company         TEXT    NOT NULL UNIQUE,
    short_code      TEXT,                                   -- card badge + API path key, e.g. "AMZ"; derived from company if unset
    owner           TEXT,                                    -- internal owner/CSM managing the account
    primary_contact TEXT,
    status          TEXT    NOT NULL DEFAULT 'in_progress',  -- in_progress | signed | freezer
    phase           TEXT    NOT NULL DEFAULT '',              -- free-text current phase, e.g. "Implementation Phase"
    progress        INTEGER NOT NULL DEFAULT 0,               -- 0-100
    target_date     TEXT,                                    -- expected completion/signing date
    notes           TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_onboardings_status ON onboardings (status);
-- short_code is the API path key for a POC (see handler.go), so it must be unique.
CREATE UNIQUE INDEX IF NOT EXISTS idx_onboardings_short_code ON onboardings (short_code);

-- Activity log — calls/emails/chats logged against a POC over time.
CREATE TABLE IF NOT EXISTS onboarding_logs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    onboarding_id  INTEGER NOT NULL REFERENCES onboardings(id) ON DELETE CASCADE,
    contact_date   TEXT    NOT NULL,                 -- date the contact happened
    contact_type   TEXT    NOT NULL DEFAULT 'call',  -- call | email | chat
    reached_by     TEXT,                             -- internal person who reached out
    contact_person TEXT,                             -- external person contacted
    description    TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_onboarding_logs_onboarding ON onboarding_logs (onboarding_id);
