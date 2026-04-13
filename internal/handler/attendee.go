package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	"github.com/James-Hou22/pager/internal/store"
)

type publicEvent struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	WelcomeDescription *string `json:"welcome_description"`
	Status             string  `json:"status"`
}

type publicChannel struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
}

// GET /attendee/events/:accessCode
// Public — no auth required. :accessCode is the event's opaque public token.
// Response 200: publicEvent JSON
func (h *Handler) getPublicEvent(c *fiber.Ctx) error {
	accessCode := c.Params("eventId")

	event, err := h.store.GetEventByAccessCode(c.Context(), accessCode)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
		}
		log.Printf("getPublicEvent: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(publicEvent{
		ID:                 event.ID,
		Name:               event.Name,
		WelcomeDescription: event.WelcomeDescription,
		Status:             string(event.Status),
	})
}

// GET /attendee/events/:accessCode/channels
// Public — no auth required. :accessCode is the event's opaque public token.
// Response 200: []publicChannel JSON
func (h *Handler) getPublicChannels(c *fiber.Ctx) error {
	accessCode := c.Params("eventId")

	event, err := h.store.GetEventByAccessCode(c.Context(), accessCode)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
		}
		log.Printf("getPublicChannels: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	channels, err := h.store.GetChannelsByEventID(c.Context(), event.ID)
	if err != nil {
		log.Printf("getPublicChannels: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	result := make([]publicChannel, len(channels))
	for i, ch := range channels {
		result[i] = publicChannel{
			ID:          ch.ID,
			Name:        ch.Name,
			Description: ch.Description,
			Status:      string(ch.Status),
		}
	}
	return c.JSON(result)
}

// POST /channel/:id/sub
// Public — no auth required.
// Body: Web Push subscription JSON from the browser.
// Response 200: {"token":"..."}
// If a session already exists for the push endpoint, the existing token is returned.
func (h *Handler) subscribeChannel(c *fiber.Ctx) error {
	channelID := c.Params("id")
	body := c.Body()

	var sub struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(body, &sub); err != nil || sub.Endpoint == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid subscription"})
	}

	if err := h.store.AddSubscriber(c.Context(), channelID, string(body)); err != nil {
		log.Printf("subscribeChannel: AddSubscriber: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	token, err := h.store.GetAttendeeSessionByEndpoint(c.Context(), sub.Endpoint)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			log.Printf("subscribeChannel: GetAttendeeSessionByEndpoint: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		token = uuid.NewString()
		if err := h.store.CreateAttendeeSession(c.Context(), token, sub.Endpoint); err != nil {
			log.Printf("subscribeChannel: CreateAttendeeSession: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
	}

	return c.JSON(fiber.Map{"token": token})
}

// GET /attendee/channel/:channelId/messages
// Requires X-Attendee-Token header. Returns messages for the channel in
// chronological order as a JSON array of {text, channel_id, sent_at}.
func (h *Handler) getAttendeeChannelMessages(c *fiber.Ctx) error {
	token := c.Get("X-Attendee-Token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
	}
	if _, err := h.store.GetAttendeeSessionByToken(c.Context(), token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		log.Printf("getAttendeeChannelMessages: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	channelID := c.Params("channelId")
	msgs, err := h.store.GetMessages(c.Context(), channelID)
	if err != nil {
		log.Printf("getAttendeeChannelMessages: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	type item struct {
		Text      string `json:"text"`
		ChannelID string `json:"channel_id"`
		SentAt    string `json:"sent_at"`
	}
	result := make([]item, len(msgs))
	for i, m := range msgs {
		var sentAt string
		if m.SentAt != nil {
			sentAt = m.SentAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		result[i] = item{
			Text:      m.Body,
			ChannelID: m.ChannelID,
			SentAt:    sentAt,
		}
	}
	return c.JSON(result)
}

// GET /manifest/:accessCode
// Public — no auth required.
// Returns a Web App Manifest scoped to the specific event so iOS labels
// the home screen icon with the event name and opens the correct start URL.
func (h *Handler) getManifest(c *fiber.Ctx) error {
	accessCode := c.Params("accessCode")

	event, err := h.store.GetEventByAccessCode(c.Context(), accessCode)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
		}
		log.Printf("getManifest: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	c.Set("Content-Type", "application/manifest+json")
	return c.JSON(fiber.Map{
		"name":             event.Name,
		"short_name":       event.Name,
		"start_url":        "/event/" + event.AccessCode,
		"display":          "standalone",
		"background_color": "#0f0f0f",
		"theme_color":      "#6c47ff",
		"icons": []fiber.Map{
			{"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"},
			{"src": "/icon-512.png", "sizes": "512x512", "type": "image/png"},
		},
	})
}

// GET /channel/:id/sse
// Public — no auth required.
// Streams Redis pub/sub messages to the client as Server-Sent Events.
func (h *Handler) sseChannel(c *fiber.Ctx) error {
	channelID := c.Params("id")

	if _, err := h.store.GetChannelByID(c.Context(), channelID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		log.Printf("sseChannel: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	pubsub := h.store.Subscribe(c.Context(), channelID)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer pubsub.Close()
		for msg := range pubsub.Channel() {
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			if err := w.Flush(); err != nil {
				return
			}
		}
	}))

	return nil
}

func (h *Handler) verifyAttendeeToken(c *fiber.Ctx) error {
	token := c.Get("X-Attendee-Token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
	}
	if _, err := h.store.GetAttendeeSessionByToken(c.Context(), token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		log.Printf("verifyAttendeeToken: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"valid": true})
}
