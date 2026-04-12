package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Event maps to the events table.
type Event struct {
	ID                 string
	OrganizerID        string
	Name               string
	AccessCode         string
	Status             EventStatus
	WelcomeDescription *string
	StartsAt           *time.Time
	EndsAt             *time.Time
	CreatedAt          time.Time
}

// GetEventByID fetches a single event by its UUID.
// Returns ErrNotFound if no row matches.
func (s *Store) GetEventByID(ctx context.Context, eventID string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`SELECT id, organizer_id, name, access_code, status, welcome_description, starts_at, ends_at, created_at
		 FROM events WHERE id = $1`,
		eventID,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.WelcomeDescription, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("store.GetEventByID %s: %w", eventID, ErrNotFound)
		}
		return Event{}, fmt.Errorf("store.GetEventByID: %w", err)
	}
	return e, nil
}

// GetEventByAccessCode fetches a single event by its public access code.
// Returns ErrNotFound if no row matches.
func (s *Store) GetEventByAccessCode(ctx context.Context, accessCode string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`SELECT id, organizer_id, name, access_code, status, welcome_description, starts_at, ends_at, created_at
		 FROM events WHERE access_code = $1`,
		accessCode,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.WelcomeDescription, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("store.GetEventByAccessCode %s: %w", accessCode, ErrNotFound)
		}
		return Event{}, fmt.Errorf("store.GetEventByAccessCode: %w", err)
	}
	return e, nil
}

// GetEventsByOrganizerID returns all events for an organizer, newest first.
// Returns an empty slice if none are found.
func (s *Store) GetEventsByOrganizerID(ctx context.Context, organizerID string) ([]Event, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, organizer_id, name, access_code, status, welcome_description, starts_at, ends_at, created_at
		 FROM events WHERE organizer_id = $1
		 ORDER BY created_at DESC`,
		organizerID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetEventsByOrganizerID: %w", err)
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.WelcomeDescription, &e.StartsAt, &e.EndsAt, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store.GetEventsByOrganizerID: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.GetEventsByOrganizerID: %w", err)
	}
	return events, nil
}

// CreateEvent inserts a new event with status EventStatusDraft and returns the created row.
// startsAt, endsAt, and welcomeDescription are optional — pass nil to leave them unset.
func (s *Store) CreateEvent(ctx context.Context, organizerID, name string, welcomeDescription *string, startsAt, endsAt *time.Time) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`INSERT INTO events (organizer_id, name, access_code, status, welcome_description, starts_at, ends_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, organizer_id, name, access_code, status, welcome_description, starts_at, ends_at, created_at`,
		organizerID, name, uuid.NewString(), EventStatusDraft, welcomeDescription, startsAt, endsAt,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.WelcomeDescription, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		return Event{}, fmt.Errorf("store.CreateEvent: %w", err)
	}
	return e, nil
}

// UpdateEventStatus sets the status of an event and returns the updated row.
// Returns ErrNotFound if no event with that ID exists.
func (s *Store) UpdateEventStatus(ctx context.Context, eventID string, status EventStatus) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`UPDATE events SET status = $1 WHERE id = $2
		 RETURNING id, organizer_id, name, access_code, status, welcome_description, starts_at, ends_at, created_at`,
		status, eventID,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.WelcomeDescription, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("store.UpdateEventStatus %s: %w", eventID, ErrNotFound)
		}
		return Event{}, fmt.Errorf("store.UpdateEventStatus: %w", err)
	}
	return e, nil
}

// GetEventByChannelID fetches the parent event of a channel by joining through event_id.
// Returns ErrNotFound if the channel or its event does not exist.
func (s *Store) GetEventByChannelID(ctx context.Context, channelID string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`SELECT e.id, e.organizer_id, e.name, e.access_code, e.status,
		        e.welcome_description, e.starts_at, e.ends_at, e.created_at
		 FROM events e
		 JOIN channels c ON c.event_id = e.id
		 WHERE c.id = $1`,
		channelID,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.WelcomeDescription, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("store.GetEventByChannelID %s: %w", channelID, ErrNotFound)
		}
		return Event{}, fmt.Errorf("store.GetEventByChannelID: %w", err)
	}
	return e, nil
}
