# Pager — Broadcast Notification Tool

## What this is
A lightweight event broadcast tool. Organizers create events with channels,
attendees subscribe via QR code, organizer blasts one-way push notifications.

## Architecture
- Go + Fiber for HTTP
- Redis for real-time fan-out (pub/sub, subscriber sets, message cache)
- PostgreSQL for durable storage (organizers, events, channels, messages, attendee sessions)
- Web Push (VAPID) for lock-screen notifications
- SSE as in-browser fallback
- React + Vite frontend (built output served from web/static/)

## Project structure
- cmd/pager/main.go        — entry point, wires everything together
- internal/store/          — all Redis and PostgreSQL access
- internal/push/           — web push fan-out (bounded worker pool, 50 workers)
- internal/handler/        — Fiber HTTP route handlers
- internal/middleware/      — JWT auth middleware
- migrations/              — PostgreSQL migration SQL files
- frontend/                — React + Vite source (builds to web/static/)
- web/static/              — built PWA frontend served by Fiber

## PostgreSQL schema
- organizers     — accounts (email, password_hash)
- events         — belong to organizers, identified by access_code
- channels       — belong to events, have status (inactive/active/closed)
- messages       — belong to channels, have body and optional scheduled_at
- attendee_sessions — map push endpoint to session token

## Redis key schema
- channel:{id}             → hash (name, created_at)
- channel:{id}:subs        → set of web push subscription JSON
- channel:{id}:messages    → list (capped at 50)
- channel:{id}:events      → pub/sub channel for SSE fan-out

## Rules
- No global state. Dependencies injected via structs.
- All Redis keys must use the helpers defined in store/store.go
- Errors must be wrapped with context: fmt.Errorf("store.FuncName: %w", err)
- No logic in handlers — handlers call into internal packages only
- Fan-out must use a worker pool, never unbounded goroutines

## Environment variables
See .env.example for all required vars. Never hardcode secrets.
Key vars: POSTGRES_URL, REDIS_URL, JWT_SECRET, VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, VAPID_EMAIL
