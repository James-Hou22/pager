package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Channel maps to the channels table.
type Channel struct {
	ID          string
	EventID     string
	Name        string
	Description *string
	RedisKey    string
	OpensAt     *time.Time
	ClosesAt    *time.Time
	CreatedAt   time.Time
}

// CreateChannel inserts a new channel into Postgres and initialises the
// corresponding Redis hash so real-time operations work immediately.
func (s *Store) CreateChannel(ctx context.Context, eventID, name string, description *string, opensAt, closesAt *time.Time) (Channel, error) {
	id := uuid.NewString()

	var ch Channel
	err := s.db.QueryRow(ctx,
		`INSERT INTO channels (id, event_id, name, description, redis_key, opens_at, closes_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, event_id, name, description, redis_key, opens_at, closes_at, created_at`,
		id, eventID, name, description, id, opensAt, closesAt,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.Description, &ch.RedisKey, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
	if err != nil {
		return Channel{}, fmt.Errorf("store.CreateChannel: %w", err)
	}

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
		`SELECT id, event_id, name, description, redis_key, opens_at, closes_at, created_at
		 FROM channels WHERE id = $1`,
		channelID,
	).Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.Description, &ch.RedisKey, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt)
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
		`SELECT id, event_id, name, description, redis_key, opens_at, closes_at, created_at
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
		if err := rows.Scan(&ch.ID, &ch.EventID, &ch.Name, &ch.Description, &ch.RedisKey, &ch.OpensAt, &ch.ClosesAt, &ch.CreatedAt); err != nil {
			return nil, fmt.Errorf("store.GetChannelsByEventID: %w", err)
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.GetChannelsByEventID: %w", err)
	}
	return channels, nil
}

// RemoveSubscriber removes a specific web push subscription from the channel's subscriber set.
func (s *Store) RemoveSubscriber(ctx context.Context, channelID, subJSON string) error {
	if err := s.rdb.SRem(ctx, channelSubsKey(channelID), subJSON).Err(); err != nil {
		return fmt.Errorf("store.RemoveSubscriber: %w", err)
	}
	return nil
}

// AddSubscriber adds a web push subscription JSON string to the channel's subscriber set.
func (s *Store) AddSubscriber(ctx context.Context, channelID, subJSON string) error {
	if err := s.rdb.SAdd(ctx, channelSubsKey(channelID), subJSON).Err(); err != nil {
		return fmt.Errorf("store.AddSubscriber: %w", err)
	}
	return nil
}

// GetSubscribers returns all web push subscription JSON strings for a channel.
func (s *Store) GetSubscribers(ctx context.Context, channelID string) ([]string, error) {
	subs, err := s.rdb.SMembers(ctx, channelSubsKey(channelID)).Result()
	if err != nil {
		return nil, fmt.Errorf("store.GetSubscribers: %w", err)
	}
	return subs, nil
}

// Publish sends a message on the channel's pub/sub event stream.
func (s *Store) Publish(ctx context.Context, channelID, message string) error {
	if err := s.rdb.Publish(ctx, channelEventsKey(channelID), message).Err(); err != nil {
		return fmt.Errorf("store.Publish: %w", err)
	}
	return nil
}

// Subscribe returns a pub/sub subscription for the channel's event stream.
// The caller is responsible for closing the returned PubSub.
func (s *Store) Subscribe(ctx context.Context, channelID string) PubSub {
	return s.rdb.Subscribe(ctx, channelEventsKey(channelID))
}
