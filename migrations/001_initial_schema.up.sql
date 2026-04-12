CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE TABLE public.organizers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (email)
);

CREATE TABLE public.events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    organizer_id uuid NOT NULL REFERENCES public.organizers(id) ON DELETE CASCADE,
    name text NOT NULL,
    access_code text NOT NULL,
    status text DEFAULT 'draft' NOT NULL,
    starts_at timestamp with time zone,
    ends_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (access_code),
    CONSTRAINT events_status_check CHECK (status = ANY (ARRAY['draft', 'active', 'closed']))
);

CREATE TABLE public.channels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_id uuid NOT NULL REFERENCES public.events(id) ON DELETE CASCADE,
    name text NOT NULL,
    redis_key text NOT NULL,
    status text DEFAULT 'inactive' NOT NULL,
    opens_at timestamp with time zone,
    closes_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (redis_key),
    CONSTRAINT channels_status_check CHECK (status = ANY (ARRAY['inactive', 'active', 'closed']))
);

CREATE TABLE public.messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    channel_id uuid NOT NULL REFERENCES public.channels(id) ON DELETE CASCADE,
    body text,
    content jsonb,
    scheduled_at timestamp with time zone,
    sent_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX idx_events_organizer_id ON public.events(organizer_id);
CREATE INDEX idx_channels_event_id ON public.channels(event_id);
CREATE INDEX idx_messages_channel_id ON public.messages(channel_id);
CREATE INDEX idx_messages_scheduled_at ON public.messages(scheduled_at) WHERE sent_at IS NULL;
