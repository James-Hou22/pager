package handler

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/James-Hou22/pager/internal/store"
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
		Name               string `json:"name"`
		WelcomeDescription string `json:"welcome_description"`
		StartsAt           string `json:"starts_at"`
		EndsAt             string `json:"ends_at"`
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

	var welcomeDescription *string
	if body.WelcomeDescription != "" {
		welcomeDescription = &body.WelcomeDescription
	}

	event, err := h.store.CreateEvent(c.Context(), organizerID, body.Name, welcomeDescription, &startsAtVal, &endsAtVal)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(event)
}

// PATCH /events/:eventId/status
// Authorization: Bearer <token>
// Body: {"status":"draft"|"active"|"closed"}
// Response 200: updated Event as JSON
func (h *Handler) updateEventStatus(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
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

	var status store.EventStatus
	switch store.EventStatus(body.Status) {
	case store.EventStatusDraft, store.EventStatusActive, store.EventStatusClosed:
		status = store.EventStatus(body.Status)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	event, err := h.store.UpdateEventStatus(c.Context(), eventID, status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
		}
		log.Printf("updateEventStatus: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(event)
}
