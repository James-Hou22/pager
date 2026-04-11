package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// GET /events
// Authorization: Bearer <token>
// Response 200: JSON array of events ([] if none)
func (h *Handler) listEvents(c *fiber.Ctx) error {
	organizerID, _ := c.Locals("organizer_id").(string)

	events, err := h.store.GetEventsByOrganizerID(c.Context(), organizerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(events)
}

// POST /events
// Authorization: Bearer <token>
// Body: {"name":"...","starts_at":"2026-04-15T18:00:00Z","ends_at":"2026-04-15T22:00:00Z"}
// starts_at and ends_at are optional ISO 8601 strings.
// Response 201: created Event as JSON
func (h *Handler) createEvent(c *fiber.Ctx) error {
	organizerID, _ := c.Locals("organizer_id").(string)

	var body struct {
		Name     string `json:"name"`
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	if body.StartsAt == "" || body.EndsAt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "starts_at and ends_at are required"})
	}

	startsAtVal, err := time.Parse(time.RFC3339, body.StartsAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "starts_at must be ISO 8601"})
	}
	endsAtVal, err := time.Parse(time.RFC3339, body.EndsAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ends_at must be ISO 8601"})
	}

	event, err := h.store.CreateEvent(c.Context(), organizerID, body.Name, &startsAtVal, &endsAtVal)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(event)
}
