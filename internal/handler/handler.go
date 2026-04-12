package handler

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/James-Hou22/pager/internal/middleware"
	"github.com/James-Hou22/pager/internal/push"
	"github.com/James-Hou22/pager/internal/store"
)

type Handler struct {
	store     *store.Store
	fanout    *push.Fanout
	jwtSecret []byte
}

func New(s *store.Store, f *push.Fanout, jwtSecret []byte) *Handler {
	return &Handler{store: s, fanout: f, jwtSecret: jwtSecret}
}

func (h *Handler) Register(app *fiber.App) {
	app.Post("/auth/register", h.authRegister)
	app.Post("/auth/login", h.authLogin)
	app.Get("/auth/me", middleware.Auth(h.jwtSecret), h.authMe)

	app.Get("/events", middleware.Auth(h.jwtSecret), h.listEvents)
	app.Post("/events", middleware.Auth(h.jwtSecret), h.createEvent)
	app.Patch("/events/:eventId/status", middleware.Auth(h.jwtSecret), h.updateEventStatus)
	app.Get("/events/:eventId/channels", middleware.Auth(h.jwtSecret), h.listChannels)
	app.Post("/events/:eventId/channels", middleware.Auth(h.jwtSecret), h.createChannel)
	app.Patch("/events/:eventId/channels/:channelId/status", middleware.Auth(h.jwtSecret), h.updateChannelStatus)
	app.Get("/events/:eventId/channels/:channelId/messages", middleware.Auth(h.jwtSecret), h.getChannelMessages)

	app.Post("/channel/:id/blast", middleware.Auth(h.jwtSecret), h.Blast)

	app.Get("/attendee/events/:eventId", h.getPublicEvent)
	app.Get("/attendee/events/:eventId/channels", h.getPublicChannels)
	app.Get("/attendee/channel/:channelId/messages", h.getAttendeeChannelMessages)
	app.Get("/manifest/:accessCode", h.getManifest)
	app.Post("/channel/:id/sub", h.subscribeChannel)
	app.Get("/channel/:id/sse", h.sseChannel)
}

// verifyEventOwnership fetches the event by ID and confirms the given organizerID
// owns it. On failure it writes the appropriate HTTP response and returns a
// non-nil error so the caller can do:
//
//	event, err := h.verifyEventOwnership(c, eventID, organizerID)
//	if err != nil { return err }
func (h *Handler) verifyEventOwnership(c *fiber.Ctx, eventID, organizerID string) (store.Event, error) {
	event, err := h.store.GetEventByID(c.Context(), eventID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.Event{}, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
		}
		return store.Event{}, c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if event.OrganizerID != organizerID {
		return store.Event{}, c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you do not own this event"})
	}
	return event, nil
}

// POST /channel/:id/blast
// Body: {"message":"..."}
// Stores the message and publishes it to SSE subscribers.
// Response 200
func (h *Handler) Blast(c *fiber.Ctx) error {
	id := c.Params("id")
	organizerID, _ := c.Locals("organizer_id").(string)

	var body struct {
		Message string `json:"message"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message required"})
	}

	// GetEventByChannelID returns ErrNotFound if the channel doesn't exist,
	// so no separate channel existence check is needed.
	event, err := h.store.GetEventByChannelID(c.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if event.OrganizerID != organizerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you do not own this channel"})
	}

	// Persist to Postgres first — abort if the write fails so no message is
	// delivered without a durable record.
	if _, err := h.store.CreateMessage(c.Context(), id, body.Message); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if err := h.store.AddMessage(c.Context(), id, body.Message); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if err := h.store.Publish(c.Context(), id, body.Message); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	// Fan out to web push subscribers in the background so the HTTP response
	// is not held open waiting for all sends to complete.
	go h.fanout.Send(context.Background(), id, body.Message)

	return c.SendStatus(fiber.StatusOK)
}
