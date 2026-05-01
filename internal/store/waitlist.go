package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// AddToWaitlist inserts an email into the waitlist.
// Returns a wrapped ErrConflict if the email is already registered.
func (s *Store) AddToWaitlist(ctx context.Context, email string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO waitlist (email) VALUES ($1)`,
		email,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("store.AddToWaitlist: %w", ErrConflict)
		}
		return fmt.Errorf("store.AddToWaitlist: %w", err)
	}
	return nil
}
