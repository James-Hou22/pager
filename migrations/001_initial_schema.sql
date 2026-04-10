CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE organizers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id  UUID NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    access_code   TEXT NOT NULL UNIQUE,
    status        TEXT NOT NULL DEFAULT 'draft',
    starts_at     TIMESTAMPTZ,
    ends_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE channels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    redis_key  TEXT NOT NULL UNIQUE,
    opens_at   TIMESTAMPTZ,
    closes_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE messages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id   UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    body         TEXT,
    content      JSONB,
    scheduled_at TIMESTAMPTZ,
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_organizer_id    ON events(organizer_id);
CREATE INDEX idx_channels_event_id      ON channels(event_id);
CREATE INDEX idx_messages_channel_id    ON messages(channel_id);
CREATE INDEX idx_messages_scheduled_at  ON messages(scheduled_at) WHERE sent_at IS NULL;