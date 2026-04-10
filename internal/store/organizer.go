package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Organizer maps to the organizers table.
type Organizer struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// CreateOrganizer inserts a new organizer and returns the created row.
// Returns a wrapped ErrConflict if the email is already registered.
func (s *Store) CreateOrganizer(ctx context.Context, email, passwordHash string) (Organizer, error) {
	var o Organizer
	err := s.db.QueryRow(ctx,
		`INSERT INTO organizers (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Organizer{}, fmt.Errorf("store.CreateOrganizer: %w", ErrConflict)
		}
		return Organizer{}, fmt.Errorf("store.CreateOrganizer: %w", err)
	}
	return o, nil
}

// GetOrganizerByEmail fetches an organizer by email.
// Returns ErrNotFound if no row matches.
func (s *Store) GetOrganizerByEmail(ctx context.Context, email string) (Organizer, error) {
	var o Organizer
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at
		 FROM organizers WHERE email = $1`,
		email,
	).Scan(&o.ID, &o.Email, &o.PasswordHash, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Organizer{}, fmt.Errorf("store.GetOrganizerByEmail %s: %w", email, ErrNotFound)
		}
		return Organizer{}, fmt.Errorf("store.GetOrganizerByEmail: %w", err)
	}
	return o, nil
}
