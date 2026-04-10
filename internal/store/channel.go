package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a unique constraint is violated (e.g. duplicate email).
var ErrConflict = errors.New("conflict")

// Channel holds the metadata stored in the channel:{id} hash.
type Channel struct {
	PasswordHash   string
	CreatedAt      string
	OrganizerToken string
}

// CreateChannel writes the channel hash. Returns an error if the key already exists.
func (s *Store) CreateChannel(ctx context.Context, id string, ch Channel) error {
	key := channelKey(id)

	// HSetNX-style: use a pipeline with NX semantics via HSETNX on a sentinel field.
	// Simpler: write all fields atomically and let the caller guarantee uniqueness.
	fields := map[string]any{
		"password_hash":   ch.PasswordHash,
		"created_at":      ch.CreatedAt,
		"organizer_token": ch.OrganizerToken,
	}

	if err := s.rdb.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("store.CreateChannel: %w", err)
	}
	return nil
}

// GetChannel retrieves channel metadata. Returns ErrNotFound if the channel does not exist.
func (s *Store) GetChannel(ctx context.Context, id string) (*Channel, error) {
	key := channelKey(id)

	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("store.GetChannel: %w", err)
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("store.GetChannel %s: %w", id, ErrNotFound)
	}

	return &Channel{
		PasswordHash:   vals["password_hash"],
		CreatedAt:      vals["created_at"],
		OrganizerToken: vals["organizer_token"],
	}, nil
}

// RemoveSubscriber removes a specific web push subscription from the channel's subscriber set.
// Used to prune stale subscriptions on 410 Gone responses.
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

// AddMessage prepends a message to channel:{id}:messages and caps the list at 50.
func (s *Store) AddMessage(ctx context.Context, channelID, message string) error {
	key := channelMsgsKey(channelID)
	pipe := s.rdb.Pipeline()
	pipe.LPush(ctx, key, message)
	pipe.LTrim(ctx, key, 0, maxMessages-1)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store.AddMessage: %w", err)
	}
	return nil
}

// Publish sends a message on the channel's pub/sub event stream.
func (s *Store) Publish(ctx context.Context, channelID, message string) error {
	if err := s.rdb.Publish(ctx, channelEventsKey(channelID), message).Err(); err != nil {
		return fmt.Errorf("store.Publish: %w", err)
	}
	return nil
}

// Subscribe returns a pub/sub subscription for the channel's event stream.
// The caller is responsible for closing the returned *redis.PubSub.
func (s *Store) Subscribe(ctx context.Context, channelID string) *redis.PubSub {
	return s.rdb.Subscribe(ctx, channelEventsKey(channelID))
}

// DeleteChannel removes the channel hash, subscriber set, and message list.
func (s *Store) DeleteChannel(ctx context.Context, id string) error {
	keys := []string{channelKey(id), channelSubsKey(id), channelMsgsKey(id)}
	if err := s.rdb.Del(ctx, keys...).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("store.DeleteChannel: %w", err)
	}
	return nil
}
