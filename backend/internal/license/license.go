// Package license implements SQLite-backed license monitoring for Devtron
// installations.
package license

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"central-devtron/internal/db"
)

// License mirrors a row in the `licenses` table.
type License struct {
	ID             int64      `json:"id"`
	Customer       string     `json:"customer"`
	Installation   string     `json:"installation"`
	Edition        string     `json:"edition"`
	Seats          int        `json:"seats"`
	SeatsUsed      int        `json:"seatsUsed"`
	Status         string     `json:"status"`
	IssuedAt       *time.Time `json:"issuedAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	DevtronVersion *string    `json:"devtronVersion,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// DaysToExpiry is a convenience computed field for the frontend.
func (l License) DaysToExpiry() *int {
	if l.ExpiresAt == nil {
		return nil
	}
	d := int(time.Until(*l.ExpiresAt).Hours() / 24)
	return &d
}

var ErrNotFound = errors.New("license not found")

type Store struct{ db *sql.DB }

func NewStore(sqlDB *sql.DB) *Store { return &Store{db: sqlDB} }

const cols = `id, customer, installation, edition, seats, seats_used, status,
	issued_at, expires_at, devtron_version, notes, created_at, updated_at`

func scan(row db.RowScanner) (License, error) {
	var l License
	var issued, expires, version, notes sql.NullString
	var created, updated string
	err := row.Scan(&l.ID, &l.Customer, &l.Installation, &l.Edition, &l.Seats,
		&l.SeatsUsed, &l.Status, &issued, &expires, &version, &notes, &created, &updated)
	if err != nil {
		return l, err
	}
	l.IssuedAt = db.ParseTimePtr(issued)
	l.ExpiresAt = db.ParseTimePtr(expires)
	if version.Valid {
		l.DevtronVersion = &version.String
	}
	if notes.Valid {
		l.Notes = &notes.String
	}
	l.CreatedAt = db.ParseTime(created)
	l.UpdatedAt = db.ParseTime(updated)
	return l, nil
}

func (s *Store) List(ctx context.Context, status string) ([]License, error) {
	q := `SELECT ` + cols + ` FROM licenses`
	args := []any{}
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY expires_at IS NULL, expires_at, customer`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []License{}
	for rows.Next() {
		l, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id int64) (License, error) {
	l, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM licenses WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return License{}, ErrNotFound
	}
	return l, err
}

// Upsert inserts or updates by (customer, installation).
func (s *Store) Upsert(ctx context.Context, l License) (License, error) {
	const q = `
		INSERT INTO licenses (customer, installation, edition, seats, seats_used,
			status, issued_at, expires_at, devtron_version, notes)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT (customer, installation) DO UPDATE SET
			edition=excluded.edition, seats=excluded.seats, seats_used=excluded.seats_used,
			status=excluded.status, issued_at=excluded.issued_at, expires_at=excluded.expires_at,
			devtron_version=excluded.devtron_version, notes=excluded.notes,
			updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		RETURNING ` + cols
	return scan(s.db.QueryRowContext(ctx, q, l.Customer, l.Installation, l.Edition,
		l.Seats, l.SeatsUsed, l.Status, db.TimeArg(l.IssuedAt), db.TimeArg(l.ExpiresAt),
		db.StrArg(l.DevtronVersion), db.StrArg(l.Notes)))
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM licenses WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Summary aggregates license health for the dashboard card.
type Summary struct {
	Total          int            `json:"total"`
	ByStatus       map[string]int `json:"byStatus"`
	ExpiringSoon   int            `json:"expiringSoon"` // within 30 days
	Expired        int            `json:"expired"`
	TotalSeats     int            `json:"totalSeats"`
	TotalSeatsUsed int            `json:"totalSeatsUsed"`
}

func (s *Store) Summary(ctx context.Context) (Summary, error) {
	all, err := s.List(ctx, "")
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{ByStatus: map[string]int{}}
	now := time.Now()
	for _, l := range all {
		sum.Total++
		sum.ByStatus[l.Status]++
		sum.TotalSeats += l.Seats
		sum.TotalSeatsUsed += l.SeatsUsed
		if l.ExpiresAt != nil {
			if l.ExpiresAt.Before(now) {
				sum.Expired++
			} else if l.ExpiresAt.Before(now.Add(30 * 24 * time.Hour)) {
				sum.ExpiringSoon++
			}
		}
	}
	return sum, nil
}
