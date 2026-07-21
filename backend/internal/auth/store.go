package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"central-devtron/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// User is an application user (dashboard operator).
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Disabled     bool      `json:"-"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

var ErrNotFound = errors.New("user not found")

type Store struct{ db *sql.DB }

func NewStore(sqlDB *sql.DB) *Store { return &Store{db: sqlDB} }

func scanUser(row db.RowScanner) (User, error) {
	var u User
	var name sql.NullString
	var disabled int
	var created string
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &name, &u.Role, &disabled, &created)
	if err != nil {
		return User{}, err
	}
	if name.Valid {
		u.Name = name.String
	}
	u.Disabled = disabled != 0
	u.CreatedAt = db.ParseTime(created)
	return u, nil
}

func (s *Store) byEmail(ctx context.Context, email string) (User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, role, disabled, created_at
		 FROM users WHERE email = ?`, normalize(email)))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) byID(ctx context.Context, id int64) (User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, role, disabled, created_at
		 FROM users WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// SeedAdmin ensures an admin user exists / has the given password. It is safe to
// run on every startup: it upserts the password hash for the configured email.
func (s *Store) SeedAdmin(ctx context.Context, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	if len(password) < 8 {
		return errors.New("ADMIN_PASSWORD must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, name, role)
		VALUES (?, ?, 'Administrator', 'admin')
		ON CONFLICT (email) DO UPDATE
		  SET password_hash = excluded.password_hash, disabled = 0,
		      updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
		normalize(email), string(hash))
	return err
}

// HasAnyUser reports whether at least one user exists.
func (s *Store) HasAnyUser(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n > 0, err
}

func normalize(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
