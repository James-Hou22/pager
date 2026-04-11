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
	Status      EventStatus
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
	Status    ChannelStatus
	OpensAt   *time.Time
	ClosesAt  *time.Time
	CreatedAt time.Time
}

// GetEventByID fetches a single event by its UUID.
// Returns ErrNotFound if no row matches.
func (s *Store) GetEventByID(ctx context.Context, eventID string) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`SELECT id, organizer_id, name, access_code, status, starts_at, ends_at, created_at
		 FROM events WHERE id = $1`,
		eventID,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("store.GetEventByID %s: %w", eventID, ErrNotFound)
		}
		return Event{}, fmt.Errorf("store.GetEventByID: %w", err)
	}
	return e, nil
}

// GetEventsByOrganizerID returns all events for an organizer, newest first.
// Returns an empty slice if none are found.
func (s *Store) GetEventsByOrganizerID(ctx context.Context, organizerID string) ([]Event, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, organizer_id, name, access_code, status, starts_at, ends_at, created_at
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
		if err := rows.Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.StartsAt, &e.EndsAt, &e.CreatedAt); err != nil {
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
// startsAt and endsAt are optional — pass nil to leave them unset.
func (s *Store) CreateEvent(ctx context.Context, organizerID, name string, startsAt, endsAt *time.Time) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`INSERT INTO events (organizer_id, name, access_code, status, starts_at, ends_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, organizer_id, name, access_code, status, starts_at, ends_at, created_at`,
		organizerID, name, uuid.NewString(), EventStatusDraft, startsAt, endsAt,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		return Event{}, fmt.Errorf("store.CreateEvent: %w", err)
	}
	return e, nil
}

// CreateChannel inserts a new channel into Postgres (using the channel UUID as redis_key)
// and initialises the corresponding Redis hash so real-time operations work without
// any additional setup by the caller.
func (s *Store) CreateChannel(ctx context.Context, eventID, name string, status ChannelStatus, opensAt, closesAt *time.Time) (Channel, error) {
	id := uuid.NewString()

	var ch Channel
	err := s.db.QueryRow(ctx,
		`INSERT INTO channels (id, event_id, name, redis_key, status, opens_at, closes_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, event_id, name, redis_key, status, opens_at, closes_at, created_at`,
		id, eventID, name, id, status, opensAt, closesAt,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.RedisKey, &ch.Status, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
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
		`SELECT id, event_id, name, redis_key, status, opens_at, closes_at, created_at
		 FROM channels WHERE id = $1`,
		channelID,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.RedisKey, &ch.Status, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Channel{}, fmt.Errorf("store.GetChannelByID %s: %w", channelID, ErrNotFound)
		}
		return Channel{}, fmt.Errorf("store.GetChannelByID: %w", err)
	}
	return ch, nil
}

// GetChannelsByEventID returns all channels for an event, newest first.
// Returns an empty slice if none are found.
func (s *Store) GetChannelsByEventID(ctx context.Context, eventID string) ([]Channel, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, event_id, name, redis_key, status, opens_at, closes_at, created_at
		 FROM channels WHERE event_id = $1
		 ORDER BY created_at DESC`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetChannelsByEventID: %w", err)
	}
	defer rows.Close()

	channels := []Channel{}
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.RedisKey, &ch.Status, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt); err != nil {
			return nil, fmt.Errorf("store.GetChannelsByEventID: %w", err)
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.GetChannelsByEventID: %w", err)
	}
	return channels, nil
}

// UpdateChannelStatus sets the status of a channel and returns the updated row.
// Returns ErrNotFound if no channel with that ID exists.
func (s *Store) UpdateChannelStatus(ctx context.Context, channelID string, status ChannelStatus) (Channel, error) {
	var ch Channel
	err := s.db.QueryRow(ctx,
		`UPDATE channels SET status = $1 WHERE id = $2
		 RETURNING id, event_id, name, redis_key, status, opens_at, closes_at, created_at`,
		status, channelID,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.RedisKey, &ch.Status, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Channel{}, fmt.Errorf("store.UpdateChannelStatus %s: %w", channelID, ErrNotFound)
		}
		return Channel{}, fmt.Errorf("store.UpdateChannelStatus: %w", err)
	}
	return ch, nil
}

// UpdateEventStatus sets the status of an event and returns the updated row.
// Returns ErrNotFound if no event with that ID exists.
func (s *Store) UpdateEventStatus(ctx context.Context, eventID string, status EventStatus) (Event, error) {
	var e Event
	err := s.db.QueryRow(ctx,
		`UPDATE events SET status = $1 WHERE id = $2
		 RETURNING id, organizer_id, name, access_code, status, starts_at, ends_at, created_at`,
		status, eventID,
	).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.AccessCode, &e.Status, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
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
