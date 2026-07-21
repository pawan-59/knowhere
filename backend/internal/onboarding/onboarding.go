// Package onboarding implements SQLite-backed customer onboarding tracking.
package onboarding

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"central-devtron/internal/db"
)

// Onboarding mirrors a row in the `onboardings` table.
type Onboarding struct {
	ID          int64      `json:"id"`
	Customer    string     `json:"customer"`
	Owner       *string    `json:"owner,omitempty"`
	Stage       string     `json:"stage"`  // kickoff | provisioning | integration | training | live
	Status      string     `json:"status"` // on_track | at_risk | blocked | completed
	Progress    int        `json:"progress"`
	StartedAt   time.Time  `json:"startedAt"`
	TargetDate  *time.Time `json:"targetDate,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	// BlockedReason describes where a "Blocked On …" stage is blocked.
	BlockedReason *string   `json:"blockedReason,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

var ErrNotFound = errors.New("onboarding not found")

type Store struct{ db *sql.DB }

func NewStore(sqlDB *sql.DB) *Store { return &Store{db: sqlDB} }

const cols = `id, customer, owner, stage, status, progress, started_at,
	target_date, completed_at, notes, blocked_reason, created_at, updated_at`

func scan(row db.RowScanner) (Onboarding, error) {
	var o Onboarding
	var owner, notes, blockedReason sql.NullString
	var target, completed sql.NullString
	var started, created, updated string
	err := row.Scan(&o.ID, &o.Customer, &owner, &o.Stage, &o.Status, &o.Progress,
		&started, &target, &completed, &notes, &blockedReason, &created, &updated)
	if err != nil {
		return o, err
	}
	if owner.Valid {
		o.Owner = &owner.String
	}
	if notes.Valid {
		o.Notes = &notes.String
	}
	if blockedReason.Valid {
		o.BlockedReason = &blockedReason.String
	}
	o.StartedAt = db.ParseTime(started)
	o.TargetDate = db.ParseTimePtr(target)
	o.CompletedAt = db.ParseTimePtr(completed)
	o.CreatedAt = db.ParseTime(created)
	o.UpdatedAt = db.ParseTime(updated)
	return o, nil
}

func (s *Store) List(ctx context.Context, stage, status string) ([]Onboarding, error) {
	q := `SELECT ` + cols + ` FROM onboardings WHERE 1=1`
	args := []any{}
	if stage != "" {
		q += ` AND stage = ?`
		args = append(args, stage)
	}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY target_date IS NULL, target_date, customer`

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

func (s *Store) Get(ctx context.Context, id int64) (Onboarding, error) {
	o, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM onboardings WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Onboarding{}, ErrNotFound
	}
	return o, err
}

// Upsert inserts or updates by customer.
func (s *Store) Upsert(ctx context.Context, o Onboarding) (Onboarding, error) {
	const q = `
		INSERT INTO onboardings (customer, owner, stage, status, progress, target_date, completed_at, notes, blocked_reason)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT (customer) DO UPDATE SET
			owner=excluded.owner, stage=excluded.stage, status=excluded.status,
			progress=excluded.progress, target_date=excluded.target_date,
			completed_at=excluded.completed_at, notes=excluded.notes,
			blocked_reason=excluded.blocked_reason,
			updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		RETURNING ` + cols
	return scan(s.db.QueryRowContext(ctx, q, o.Customer, db.StrArg(o.Owner), o.Stage, o.Status,
		o.Progress, db.TimeArg(o.TargetDate), db.TimeArg(o.CompletedAt), db.StrArg(o.Notes), db.StrArg(o.BlockedReason)))
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM onboardings WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Summary aggregates onboarding health for the dashboard card.
type Summary struct {
	Total       int            `json:"total"`
	ByStage     map[string]int `json:"byStage"`
	ByStatus    map[string]int `json:"byStatus"`
	AtRisk      int            `json:"atRisk"`
	Blocked     int            `json:"blocked"`
	Completed   int            `json:"completed"`
	AvgProgress int            `json:"avgProgress"`
}

func (s *Store) Summary(ctx context.Context) (Summary, error) {
	all, err := s.List(ctx, "", "")
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{ByStage: map[string]int{}, ByStatus: map[string]int{}}
	totalProgress := 0
	for _, o := range all {
		sum.Total++
		sum.ByStage[o.Stage]++
		sum.ByStatus[o.Status]++
		totalProgress += o.Progress
		switch o.Status {
		case "at_risk":
			sum.AtRisk++
		case "blocked":
			sum.Blocked++
		case "completed":
			sum.Completed++
		}
	}
	if sum.Total > 0 {
		sum.AvgProgress = totalProgress / sum.Total
	}
	return sum, nil
}
