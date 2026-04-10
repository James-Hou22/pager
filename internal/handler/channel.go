package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

// POST /events/:eventId/channels
// Authorization: Bearer <token>
// Body: {"name":"..."}
// Response 201: created Channel as JSON
func (h *Handler) createChannel(c *fiber.Ctx) error {
	eventID := c.Params("eventId")

	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	channel, err := h.store.CreateChannel(c.Context(), eventID, body.Name)
	if err != nil {
		log.Printf("createChannel: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(channel)
}
