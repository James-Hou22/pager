package handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/valyala/fasthttp"
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
	app.Get("/channel/:id", h.GetChannel)
	app.Get("/channel/:id/qr", h.QRCode)
	app.Post("/channel/:id/sub", h.AddSubscriber)
	app.Post("/channel/:id/blast", middleware.Auth(h.jwtSecret), h.Blast)
	app.Get("/channel/:id/sse", h.SSE)

	app.Post("/auth/register", h.authRegister)
	app.Post("/auth/login", h.authLogin)
	app.Get("/auth/me", middleware.Auth(h.jwtSecret), h.authMe)

	app.Post("/events", middleware.Auth(h.jwtSecret), h.createEvent)
	app.Post("/events/:eventId/channels", middleware.Auth(h.jwtSecret), h.createChannel)
}

// GET /channel/:id
// Response 200: {"id":"...","created_at":"..."}
func (h *Handler) GetChannel(c *fiber.Ctx) error {
	id := c.Params("id")

	ch, err := h.store.GetRedisChannel(c.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(fiber.Map{
		"id":         id,
		"created_at": ch.CreatedAt,
	})
}

// GET /channel/:id/qr
// Returns a PNG QR code that encodes the attendee URL for this channel.
func (h *Handler) QRCode(c *fiber.Ctx) error {
	id := c.Params("id")

	if _, err := h.store.GetRedisChannel(c.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	attendeeURL := fmt.Sprintf("%s/channel/%s", baseURL(c), id)
	png, err := qrcode.Encode(attendeeURL, qrcode.Medium, 256)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(png)
}

// baseURL derives the scheme+host from the incoming request.
func baseURL(c *fiber.Ctx) string {
	scheme := "http"
	if c.Protocol() == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Hostname()
}

// POST /channel/:id/sub
// Body: web push subscription JSON (passed through as-is)
// Response 201
func (h *Handler) AddSubscriber(c *fiber.Ctx) error {
	id := c.Params("id")

	if _, err := h.store.GetRedisChannel(c.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	subJSON := string(c.Body())
	if subJSON == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subscription body required"})
	}

	if err := h.store.AddSubscriber(c.Context(), id, subJSON); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusCreated)
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

	if _, err := h.store.GetChannelByID(c.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

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

// GET /channel/:id/sse
// Streams messages as Server-Sent Events. Each blast publishes one event.
func (h *Handler) SSE(c *fiber.Ctx) error {
	id := c.Params("id")

	if _, err := h.store.GetRedisChannel(c.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Subscribe before streaming so no messages are missed after the channel check.
	pubsub := h.store.Subscribe(c.Context(), id)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer pubsub.Close()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		ch := pubsub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				// SSE comment sent as keepalive; Flush failure signals client disconnect.
				fmt.Fprintf(w, ": keepalive\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}
