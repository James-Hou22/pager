package store

import (
	"context"
	"fmt"
	"time"
)

// Message maps to the messages table.
type Message struct {
	ID        string
	ChannelID string
	Body      string
	SentAt    *time.Time
	CreatedAt time.Time
}

// CreateMessage inserts a new message into the messages table, sets sent_at to NOW(),
// and returns the created row.
func (s *Store) CreateMessage(ctx context.Context, channelID, body string) (Message, error) {
	var m Message
	err := s.db.QueryRow(ctx,
		`INSERT INTO messages (channel_id, body, sent_at)
		 VALUES ($1, $2, NOW())
		 RETURNING id, channel_id, body, sent_at, created_at`,
		channelID, body,
	).Scan(&m.ID, &m.ChannelID, &m.Body, &m.SentAt, &m.CreatedAt)
	if err != nil {
		return Message{}, fmt.Errorf("store.CreateMessage: %w", err)
	}
	return m, nil
}

// GetMessagesByChannelID returns all messages for a channel, most recent first.
// Returns an empty slice if none are found.
func (s *Store) GetMessagesByChannelID(ctx context.Context, channelID string) ([]Message, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, channel_id, body, sent_at, created_at
		 FROM messages WHERE channel_id = $1
		 ORDER BY created_at DESC`,
		channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetMessagesByChannelID: %w", err)
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Body, &m.SentAt, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("store.GetMessagesByChannelID: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.GetMessagesByChannelID: %w", err)
	}
	return messages, nil
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

// GetMessages returns all messages for the channel from Postgres in
// chronological order (oldest first), indexed by (channel_id, sent_at ASC).
func (s *Store) GetMessages(ctx context.Context, channelID string) ([]Message, error) {
	rows, err := s.db.Query(ctx,
		`SELECT body, channel_id, sent_at
		 FROM messages
		 WHERE channel_id = $1
		 ORDER BY sent_at ASC`,
		channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetMessages: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Body, &m.ChannelID, &m.SentAt); err != nil {
			return nil, fmt.Errorf("store.GetMessages scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
