ALTER TABLE channels
ADD COLUMN status TEXT NOT NULL DEFAULT 'inactive'
CHECK (status IN ('inactive', 'active', 'closed'));

ALTER TABLE events
ADD CONSTRAINT events_status_check
CHECK (status IN ('draft', 'active', 'closed'));