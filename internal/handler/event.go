package handler

import (
	"github.com/gofiber/fiber/v2"
)

// POST /events
// Authorization: Bearer <token>
// Body: {"name":"..."}
// Response 201: created Event as JSON
func (h *Handler) createEvent(c *fiber.Ctx) error {
	organizerID, _ := c.Locals("organizer_id").(string)

	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	event, err := h.store.CreateEvent(c.Context(), organizerID, body.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(event)
}
