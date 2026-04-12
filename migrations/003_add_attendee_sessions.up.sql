CREATE TABLE IF NOT EXISTS attendee_sessions (
    token         TEXT PRIMARY KEY,
    push_endpoint TEXT UNIQUE NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
