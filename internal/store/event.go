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
	ID          string
	OrganizerID string
	Name        string
	AccessCode  string
	Status      string
	StartsAt    *time.Time
	EndsAt      *time.Time
	CreatedAt   time.Time
}

// Channel maps to the channels table.
type Channel struct {
	ID        string
	EventID   string
	Name      string
	RedisKey  string
	OpensAt   *time.Time
	ClosesAt  *time.Time
	CreatedAt time.Time
}

// CreateEvent inserts a new event with status "draft" and returns the created row.
func (s *Store) CreateEvent(ctx context.Context, organizerID, name string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`INSERT INTO events (organizer_id, name, access_code, status)
		 VALUES ($1, $2, $3, 'draft')
		 RETURNING id, organizer_id, name, access_code, status, starts_at, ends_at, created_at`,
		organizerID, name, uuid.NewString(),
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		return Event{}, fmt.Errorf("store.CreateEvent: %w", err)
	}
	return e, nil
}

// CreateChannel inserts a new channel into Postgres (using the channel UUID as redis_key)
// and initialises the corresponding Redis hash so real-time operations work without
// any additional setup by the caller.
func (s *Store) CreateChannel(ctx context.Context, eventID, name string) (Channel, error) {
	id := uuid.NewString()

	var ch Channel
	err := s.db.QueryRow(ctx,
		`INSERT INTO channels (id, event_id, name, redis_key)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, event_id, name, redis_key, opens_at, closes_at, created_at`,
		id, eventID, name, id,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.RedisKey, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
	if err != nil {
		return Channel{}, fmt.Errorf("store.CreateChannel: %w", err)
	}

	// Bootstrap the Redis hash so SSE, web push, and subscriber operations
	// can reference this channel immediately after creation.
	fields := map[string]any{
		"name":       ch.Name,
		"created_at": ch.CreatedAt.UTC().Format(time.RFC3339),
	}
	if err := s.rdb.HSet(ctx, channelKey(ch.ID), fields).Err(); err != nil {
		return Channel{}, fmt.Errorf("store.CreateChannel: init redis hash: %w", err)
	}

	return ch, nil
}

// GetChannelByID fetches a channel from Postgres by its UUID.
// Returns ErrNotFound if no row matches.
func (s *Store) GetChannelByID(ctx context.Context, channelID string) (Channel, error) {
	var ch Channel
	err := s.db.QueryRow(ctx,
		`SELECT id, event_id, name, redis_key, opens_at, closes_at, created_at
		 FROM channels WHERE id = $1`,
		channelID,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.RedisKey, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Channel{}, fmt.Errorf("store.GetChannelByID %s: %w", channelID, ErrNotFound)
		}
		return Channel{}, fmt.Errorf("store.GetChannelByID: %w", err)
	}
	return ch, nil
}

// GetEventByChannelID fetches the parent event of a channel by joining through event_id.
// Returns ErrNotFound if the channel or its event does not exist.
func (s *Store) GetEventByChannelID(ctx context.Context, channelID string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`SELECT e.id, e.organizer_id, e.name, e.access_code, e.status,
		        e.starts_at, e.ends_at, e.created_at
		 FROM events e
		 JOIN channels c ON c.event_id = e.id
		 WHERE c.id = $1`,
		channelID,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("store.GetEventByChannelID %s: %w", channelID, ErrNotFound)
		}
		return Event{}, fmt.Errorf("store.GetEventByChannelID: %w", err)
	}
	return e, nil
}
