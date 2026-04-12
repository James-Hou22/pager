package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetAttendeeSessionByEndpoint returns the session token for the given push endpoint.
// Returns ErrNotFound if no session exists for that endpoint.
func (s *Store) GetAttendeeSessionByEndpoint(ctx context.Context, endpoint string) (string, error) {
	var token string
	err := s.db.QueryRow(ctx,
		`SELECT token FROM attendee_sessions WHERE push_endpoint = $1`,
		endpoint,
	).Scan(&token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("store.GetAttendeeSessionByEndpoint: %w", err)
	}
	return token, nil
}

// GetAttendeeSessionByToken returns the push endpoint for the given session token.
// Returns ErrNotFound if no session exists for that token.
func (s *Store) GetAttendeeSessionByToken(ctx context.Context, token string) (string, error) {
	var endpoint string
	err := s.db.QueryRow(ctx,
		`SELECT push_endpoint FROM attendee_sessions WHERE token = $1`,
		token,
	).Scan(&endpoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("store.GetAttendeeSessionByToken: %w", err)
	}
	return endpoint, nil
}

// CreateAttendeeSession inserts a new attendee session row.
func (s *Store) CreateAttendeeSession(ctx context.Context, token, endpoint string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO attendee_sessions (token, push_endpoint) VALUES ($1, $2)`,
		token, endpoint,
	)
	if err != nil {
		return fmt.Errorf("store.CreateAttendeeSession: %w", err)
	}
	return nil
}
