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
