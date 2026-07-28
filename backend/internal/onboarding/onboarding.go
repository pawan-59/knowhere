// Package onboarding implements SQLite-backed POC (proof-of-concept) tracking.
package onboarding

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"knowhere/internal/db"
)

// Onboarding mirrors a row in the `onboardings` table — one POC per company.
type Onboarding struct {
	ID             int64      `json:"id"`
	Company        string     `json:"company"`
	ShortCode      string     `json:"shortCode"`
	Owner          *string    `json:"owner,omitempty"`
	PrimaryContact *string    `json:"primaryContact,omitempty"`
	Status         string     `json:"status"` // in_progress | signed | freezer
	Phase          string     `json:"phase"`
	Progress       int        `json:"progress"`
	TargetDate     *time.Time `json:"targetDate,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

var ErrNotFound = errors.New("onboarding not found")

// Statuses are the only valid values for Onboarding.Status.
var Statuses = []string{"in_progress", "signed", "freezer"}

func ValidStatus(s string) bool {
	for _, v := range Statuses {
		if v == s {
			return true
		}
	}
	return false
}

// Phases are the only valid values for Onboarding.Phase — the POC pipeline steps.
var Phases = []string{
	"Discovery Call",
	"Demo Call",
	"Kickoff",
	"Success Criteria",
	"Infra Provisioning",
	"Devtron Stack Onboarding",
	"Sanity app deployed (with all needed modules)",
	"Configured Essentials Credentials",
	"Deployed 1st App",
	"Handed Over",
	"Blocked On Devtron",
	"Blocked On Client",
}

func ValidPhase(p string) bool {
	for _, v := range Phases {
		if v == p {
			return true
		}
	}
	return false
}

// ShortCodeOf derives a card badge code from a company name when one isn't
// supplied explicitly.
func ShortCodeOf(company string) string {
	c := strings.ToUpper(strings.TrimSpace(company))
	if len(c) > 3 {
		return c[:3]
	}
	return c
}

type Store struct{ db *sql.DB }

func NewStore(sqlDB *sql.DB) *Store { return &Store{db: sqlDB} }

const cols = `id, company, short_code, owner, primary_contact, status, phase, progress, target_date, notes, created_at, updated_at`

func scan(row db.RowScanner) (Onboarding, error) {
	var o Onboarding
	var shortCode, owner, contact, notes sql.NullString
	var targetDate sql.NullString
	var created, updated string
	err := row.Scan(&o.ID, &o.Company, &shortCode, &owner, &contact, &o.Status, &o.Phase, &o.Progress,
		&targetDate, &notes, &created, &updated)
	if err != nil {
		return o, err
	}
	if shortCode.Valid && shortCode.String != "" {
		o.ShortCode = shortCode.String
	} else {
		o.ShortCode = ShortCodeOf(o.Company)
	}
	if owner.Valid {
		o.Owner = &owner.String
	}
	if contact.Valid {
		o.PrimaryContact = &contact.String
	}
	if notes.Valid {
		o.Notes = &notes.String
	}
	o.TargetDate = db.ParseTimePtr(targetDate)
	o.CreatedAt = db.ParseTime(created)
	o.UpdatedAt = db.ParseTime(updated)
	return o, nil
}

// List returns POCs, optionally filtered by status and capped at limit (0 = no cap),
// most recently updated first.
func (s *Store) List(ctx context.Context, status string, limit int) ([]Onboarding, error) {
	q := `SELECT ` + cols + ` FROM onboardings WHERE 1=1`
	args := []any{}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY updated_at DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Onboarding{}
	for rows.Next() {
		o, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetByShortCode looks up a POC by its short code — the API path key used
// everywhere a client refers to a specific POC (see handler.go).
func (s *Store) GetByShortCode(ctx context.Context, shortCode string) (Onboarding, error) {
	o, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM onboardings WHERE short_code=?`, shortCode))
	if errors.Is(err, sql.ErrNoRows) {
		return Onboarding{}, ErrNotFound
	}
	return o, err
}

// uniqueShortCode finds a short code starting from base that isn't already
// used by a different company (appending 1, 2, 3... on collision), so
// auto-derived codes never clash even when company names start alike.
func (s *Store) uniqueShortCode(ctx context.Context, base, company string) (string, error) {
	code := base
	for i := 0; ; i++ {
		if i > 0 {
			code = fmt.Sprintf("%s%d", base, i)
		}
		var existingCompany string
		err := s.db.QueryRowContext(ctx, `SELECT company FROM onboardings WHERE short_code=?`, code).Scan(&existingCompany)
		if errors.Is(err, sql.ErrNoRows) || existingCompany == company {
			return code, nil
		}
		if err != nil {
			return "", err
		}
	}
}

// Upsert inserts or updates by company.
func (s *Store) Upsert(ctx context.Context, o Onboarding) (Onboarding, error) {
	shortCode := o.ShortCode
	if shortCode == "" {
		code, err := s.uniqueShortCode(ctx, ShortCodeOf(o.Company), o.Company)
		if err != nil {
			return Onboarding{}, err
		}
		shortCode = code
	}
	const q = `
		INSERT INTO onboardings (company, short_code, owner, primary_contact, status, phase, progress, target_date, notes)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT (company) DO UPDATE SET
			short_code=excluded.short_code, owner=excluded.owner, primary_contact=excluded.primary_contact,
			status=excluded.status, phase=excluded.phase, progress=excluded.progress,
			target_date=excluded.target_date, notes=excluded.notes,
			updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		RETURNING ` + cols
	return scan(s.db.QueryRowContext(ctx, q, o.Company, shortCode, db.StrArg(o.Owner), db.StrArg(o.PrimaryContact),
		o.Status, o.Phase, o.Progress, db.TimeArg(o.TargetDate), db.StrArg(o.Notes)))
}

// DeleteByShortCode removes a POC by its short code.
func (s *Store) DeleteByShortCode(ctx context.Context, shortCode string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM onboardings WHERE short_code=?`, shortCode)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Summary aggregates POC counts by status for the dashboard cards.
type Summary struct {
	Total      int `json:"total"`
	InProgress int `json:"inProgress"`
	Signed     int `json:"signed"`
	Freezer    int `json:"freezer"`
}

func (s *Store) Summary(ctx context.Context) (Summary, error) {
	all, err := s.List(ctx, "", 0)
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for _, o := range all {
		sum.Total++
		switch o.Status {
		case "in_progress":
			sum.InProgress++
		case "signed":
			sum.Signed++
		case "freezer":
			sum.Freezer++
		}
	}
	return sum, nil
}

// Log is one activity-log entry — a logged call/email/chat against a POC.
type Log struct {
	ID            int64     `json:"id"`
	OnboardingID  int64     `json:"onboardingId"`
	ContactDate   time.Time `json:"contactDate"`
	ContactType   string    `json:"contactType"` // call | email | chat
	ReachedBy     *string   `json:"reachedBy,omitempty"`
	ContactPerson *string   `json:"contactPerson,omitempty"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ContactTypes are the only valid values for Log.ContactType.
var ContactTypes = []string{"call", "email", "chat"}

func ValidContactType(t string) bool {
	for _, v := range ContactTypes {
		if v == t {
			return true
		}
	}
	return false
}

const logCols = `id, onboarding_id, contact_date, contact_type, reached_by, contact_person, description, created_at`

func scanLog(row db.RowScanner) (Log, error) {
	var l Log
	var reachedBy, contactPerson sql.NullString
	var contactDate, created string
	err := row.Scan(&l.ID, &l.OnboardingID, &contactDate, &l.ContactType, &reachedBy, &contactPerson, &l.Description, &created)
	if err != nil {
		return Log{}, err
	}
	if reachedBy.Valid {
		l.ReachedBy = &reachedBy.String
	}
	if contactPerson.Valid {
		l.ContactPerson = &contactPerson.String
	}
	l.ContactDate = db.ParseTime(contactDate)
	l.CreatedAt = db.ParseTime(created)
	return l, nil
}

// AddLog appends a log entry to the POC identified by shortCode. Returns ErrNotFound if it doesn't exist.
func (s *Store) AddLog(ctx context.Context, shortCode string, l Log) (Log, error) {
	o, err := s.GetByShortCode(ctx, shortCode)
	if err != nil {
		return Log{}, err
	}
	q := `INSERT INTO onboarding_logs (onboarding_id, contact_date, contact_type, reached_by, contact_person, description)
		VALUES (?,?,?,?,?,?)
		RETURNING ` + logCols
	return scanLog(s.db.QueryRowContext(ctx, q, o.ID, l.ContactDate.UTC().Format(time.RFC3339),
		l.ContactType, db.StrArg(l.ReachedBy), db.StrArg(l.ContactPerson), l.Description))
}

// ListLogs returns the log entries for the POC identified by shortCode, most recent contact
// first. Returns ErrNotFound if it doesn't exist.
func (s *Store) ListLogs(ctx context.Context, shortCode string) ([]Log, error) {
	o, err := s.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+logCols+` FROM onboarding_logs WHERE onboarding_id=? ORDER BY contact_date DESC, id DESC`,
		o.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Log{}
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
