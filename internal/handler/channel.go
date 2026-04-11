package handler

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/James-Hou22/pager/internal/store"
)

// GET /events/:eventId/channels
// Authorization: Bearer <token>
// Response 200: JSON array of channels ([] if none)
func (h *Handler) listChannels(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	organizerID, _ := c.Locals("organizer_id").(string)

	if _, err := h.verifyEventOwnership(c, eventID, organizerID); err != nil {
		return err
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
// Body: {"name":"...","status":"inactive"|"active","opens_at":"...","closes_at":"..."}
// status, opens_at, and closes_at are optional. status defaults to inactive.
// Response 201: created Channel as JSON
func (h *Handler) createChannel(c *fiber.Ctx) error {
	eventID := c.Params("eventId")

	var body struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		OpensAt  string `json:"opens_at"`
		ClosesAt string `json:"closes_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	status := store.ChannelStatusInactive
	if body.Status != "" {
		switch store.ChannelStatus(body.Status) {
		case store.ChannelStatusInactive, store.ChannelStatusActive:
			status = store.ChannelStatus(body.Status)
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
		}
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

	channel, err := h.store.CreateChannel(c.Context(), eventID, body.Name, status, opensAt, closesAt)
	if err != nil {
		log.Printf("createChannel: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(channel)
}

// PATCH /events/:eventId/channels/:channelId/status
// Authorization: Bearer <token>
// Body: {"status":"inactive"|"active"|"closed"}
// Response 200: updated Channel as JSON
func (h *Handler) updateChannelStatus(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	channelID := c.Params("channelId")
	organizerID, _ := c.Locals("organizer_id").(string)

	if _, err := h.verifyEventOwnership(c, eventID, organizerID); err != nil {
		return err
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	var status store.ChannelStatus
	switch store.ChannelStatus(body.Status) {
	case store.ChannelStatusInactive, store.ChannelStatusActive, store.ChannelStatusClosed:
		status = store.ChannelStatus(body.Status)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	channel, err := h.store.UpdateChannelStatus(c.Context(), channelID, status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "channel not found"})
		}
		log.Printf("updateChannelStatus: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(channel)
}

// GET /events/:eventId/channels/:channelId/messages
// Authorization: Bearer <token>
// Response 200: JSON array of messages ([] if none)
func (h *Handler) getChannelMessages(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	channelID := c.Params("channelId")
	organizerID, _ := c.Locals("organizer_id").(string)

	if _, err := h.verifyEventOwnership(c, eventID, organizerID); err != nil {
		return err
	}

	messages, err := h.store.GetMessagesByChannelID(c.Context(), channelID)
	if err != nil {
		log.Printf("getChannelMessages: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(messages)
}
