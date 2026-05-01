package handler

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GET /events/:eventId/channels
// Authorization: Bearer <token>
// Response 200: JSON array of channels ([] if none)
func (h *Handler) listChannels(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	organizerID, _ := c.Locals("organizer_id").(string)

	if _, err := h.verifyEventOwnership(c, eventID, organizerID); err != nil {
		return nil
	}

	channels, err := h.store.GetChannelsByEventID(c.Context(), eventID)
	if err != nil {
		log.Printf("listChannels: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(channels)
}

// POST /events/:eventId/channels
// Authorization: Bearer <token>
// Body: {"name":"...","description":"...","opens_at":"...","closes_at":"..."}
// opens_at and closes_at are optional.
// Response 201: created Channel as JSON
func (h *Handler) createChannel(c *fiber.Ctx) error {
	eventID := c.Params("eventId")

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		OpensAt     string `json:"opens_at"`
		ClosesAt    string `json:"closes_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	var opensAt, closesAt *time.Time
	if body.OpensAt != "" {
		t, err := time.Parse(time.RFC3339, body.OpensAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "opens_at must be ISO 8601"})
		}
		opensAt = &t
	}
	if body.ClosesAt != "" {
		t, err := time.Parse(time.RFC3339, body.ClosesAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "closes_at must be ISO 8601"})
		}
		closesAt = &t
	}

	var description *string
	if body.Description != "" {
		description = &body.Description
	}

	channel, err := h.store.CreateChannel(c.Context(), eventID, body.Name, description, opensAt, closesAt)
	if err != nil {
		log.Printf("createChannel: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(channel)
}

// GET /events/:eventId/channels/:channelId/messages
// Authorization: Bearer <token>
// Response 200: JSON array of messages ([] if none)
func (h *Handler) getChannelMessages(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	channelID := c.Params("channelId")
	organizerID, _ := c.Locals("organizer_id").(string)

	if _, err := h.verifyEventOwnership(c, eventID, organizerID); err != nil {
		return nil
	}

	messages, err := h.store.GetMessagesByChannelID(c.Context(), channelID)
	if err != nil {
		log.Printf("getChannelMessages: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(messages)
}
