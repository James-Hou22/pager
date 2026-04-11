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
