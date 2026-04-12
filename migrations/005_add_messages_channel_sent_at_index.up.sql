CREATE INDEX IF NOT EXISTS idx_messages_channel_sent_at
ON messages (channel_id, sent_at ASC);
